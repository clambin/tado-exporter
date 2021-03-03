package autoaway

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/actions"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

type AutoAway struct {
	actions.Actions

	Updates    chan *scheduler.TadoData
	Scheduler  *scheduler.Scheduler
	Rules      []*configuration.AutoAwayRule
	deviceInfo map[int]DeviceInfo
}

// Run waits for updates data from the scheduler and evaluates configured autoAway rules
func (autoAway *AutoAway) Run() {
	for tadoData := range autoAway.Updates {
		if tadoData == nil {
			break
		}
		if err := autoAway.process(tadoData); err != nil {
			log.WithField("err", err).Warning("failed to process autoAway rules")
		}
	}
}

// process sets the state of each mobileDevice, checks if any have come/left home and performs
// configured autoAway rules
func (autoAway *AutoAway) process(tadoData *scheduler.TadoData) (err error) {
	var actionList []actions.Action

	autoAway.updateInfo(tadoData)
	if actionList, err = autoAway.getActions(); err == nil {
		if err = autoAway.RunActions(actionList); err != nil {
			log.WithField("err", err).Warning("failed to set action")
		}
	}

	return
}

// updateInfo updates the state of each mobileDevice
func (autoAway *AutoAway) updateInfo(tadoData *scheduler.TadoData) {
	// If the map doesn't exist, create it
	if autoAway.deviceInfo == nil {
		autoAway.initDeviceInfo(tadoData)
	}

	// for each autoAway setting, add/update a record for the mobileDevice
	for mobileDeviceID, deviceInfo := range autoAway.deviceInfo {
		if mobileDevice, ok := tadoData.MobileDevice[mobileDeviceID]; ok {
			deviceInfo.mobileDevice = mobileDevice
			autoAway.deviceInfo[mobileDeviceID] = deviceInfo
		} else {
			continue
		}
	}
}

func (autoAway *AutoAway) initDeviceInfo(tadoData *scheduler.TadoData) {
	autoAway.deviceInfo = make(map[int]DeviceInfo)

	for _, rule := range autoAway.Rules {
		var (
			mobileDevice *tado.MobileDevice
			zone         *tado.Zone
		)
		// Rules file can contain either mobileDevice/zone ID or Name. Retrieve the ID for each of these
		// and discard any that aren't valid.  Update the mobileDevice/zone ID so we only need to do this once

		if mobileDevice = scheduler.LookupMobileDevice(tadoData, rule.MobileDeviceID, rule.MobileDeviceName); mobileDevice == nil {
			log.WithFields(log.Fields{
				"deviceID":   rule.MobileDeviceID,
				"deviceName": rule.MobileDeviceName,
			}).Warning("skipping unknown mobile device in AutoAway rule")

			continue
		}

		if zone = scheduler.LookupZone(tadoData, rule.ZoneID, rule.ZoneName); zone == nil {
			log.WithFields(log.Fields{
				"zoneID":   rule.ZoneID,
				"zoneName": rule.ZoneName,
			}).Warning("skipping unknown zone in AutoAway rule")
			continue
		}

		autoAway.deviceInfo[mobileDevice.ID] = DeviceInfo{
			mobileDevice:   *mobileDevice,
			zone:           *zone,
			rule:           *rule,
			state:          getInitialState(mobileDevice),
			activationTime: time.Now().Add(rule.WaitTime),
		}
	}
}

// getActions scans the DeviceInfo map and returns all required actions, i.e. any zones that
// need to be put in/out of Overlay mode.
func (autoAway *AutoAway) getActions() (actionList []actions.Action, err error) {
	for id, deviceInfo := range autoAway.deviceInfo {
		log.WithFields(log.Fields{
			"mobileDeviceID":   deviceInfo.mobileDevice.ID,
			"mobileDeviceName": deviceInfo.mobileDevice.Name,
			"state":            deviceInfo.state,
			"new_home":         deviceInfo.mobileDevice.Location.AtHome,
			"activation_time":  deviceInfo.activationTime,
		}).Debug("autoAwayInfo")

		if deviceInfo.mobileDevice.Location.Stale {
			log.WithFields(log.Fields{
				"mobileDeviceID":   deviceInfo.mobileDevice.ID,
				"mobileDeviceName": deviceInfo.mobileDevice.Name,
			}).Info("stale location. Skipping ...")
			continue
		}

		// if the mobile phone is now home but was away
		if deviceInfo.cameHome() {
			// mark the phone at home
			deviceInfo.state = autoAwayStateHome
			autoAway.deviceInfo[id] = deviceInfo
			// add action to disable the overlay
			actionList = append(actionList, actions.Action{
				Overlay: false,
				ZoneID:  deviceInfo.zone.ID,
			})
			log.WithFields(log.Fields{
				"MobileDeviceID": id,
				"ZoneID":         deviceInfo.zone.ID,
			}).Info("User returned home. Removing overlay")
			// notify via slack if needed
			err = autoAway.Scheduler.Notify("",
				fmt.Sprintf("%s is home. resetting %s to auto",
					deviceInfo.mobileDevice.Name,
					deviceInfo.zone.Name,
				),
			)
		} else
		// if the mobile phone is away, mark it as such and set the activation timer
		if deviceInfo.leftHome() {
			deviceInfo.state = autoAwayStateAway
			deviceInfo.activationTime = time.Now().Add(deviceInfo.rule.WaitTime)
			autoAway.deviceInfo[id] = deviceInfo
			// notify via slack if needed
			err = autoAway.Scheduler.Notify("",
				deviceInfo.mobileDevice.Name+" is away. will set "+
					deviceInfo.zone.Name+" to manual in "+deviceInfo.rule.WaitTime.String())
		} else
		// if the mobile phone was already away, check the activation timer
		if deviceInfo.shouldReport() {

			deviceInfo.state = autoAwayStateReported
			autoAway.deviceInfo[id] = deviceInfo
			// add action to set the overlay
			actionList = append(actionList, actions.Action{
				Overlay:           true,
				ZoneID:            deviceInfo.zone.ID,
				TargetTemperature: deviceInfo.rule.TargetTemperature,
			})
			log.WithFields(log.Fields{
				"MobileDeviceID":    id,
				"ZoneID":            deviceInfo.zone.ID,
				"TargetTemperature": deviceInfo.rule.TargetTemperature,
			}).Info("User left. Setting overlay")
			// notify via slack if needed
			err = autoAway.Scheduler.Notify("",
				fmt.Sprintf("%s is away. activating manual control in zone %s",
					deviceInfo.mobileDevice.Name,
					deviceInfo.zone.Name,
				),
			)
		}
	}
	return
}
