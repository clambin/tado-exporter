package controller

import (
	"fmt"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

// runAutoAway runOverlayLimit checks if mobileDevices have come/left home and performs
// configured autoAway rules
func (controller *Controller) runAutoAway() error {
	if controller.Configuration.AutoAwayRules == nil {
		return nil
	}

	var (
		err     error
		actions []action
	)

	// update mobiles & zones for each autoAway entry
	if err = controller.updateAutoAwayInfo(); err == nil {
		// get actions for each autoAway setting
		if actions, err = controller.getAutoAwayActions(); err == nil {
			for _, action := range actions {
				// execute each action
				if err = controller.runAction(action); err != nil {
					break
				}
			}
		}
	}

	log.WithField("err", err).Debug("runAutoAway")
	return err
}

// updateAutoAwayInfo updates the mobile device & zone information for each autoAway rule.
// On exit, the map controller.AutoAwayInfo contains the up-to-date mobileDevice information
// for any mobile device mentioned in any autoAway rule.
func (controller *Controller) updateAutoAwayInfo() (err error) {
	// If the map doesn't exist, create it
	if controller.AutoAwayInfo == nil {
		controller.AutoAwayInfo = make(map[int]AutoAwayInfo)
	}

	// for each autoAway setting, add/update a record for the mobileDevice
	for _, autoAwayRule := range *controller.Configuration.AutoAwayRules {
		var (
			mobileDevice *tado.MobileDevice
			zone         *tado.Zone
		)
		// Rules file can contain either mobileDevice/zone ID or Name. Retrieve the ID for each of these
		// and discard any that aren't valid
		if mobileDevice = controller.lookupMobileDevice(autoAwayRule.MobileDeviceID, autoAwayRule.MobileDeviceName); mobileDevice == nil {
			log.WithFields(log.Fields{
				"deviceID":   autoAwayRule.MobileDeviceID,
				"deviceName": autoAwayRule.MobileDeviceName,
			}).Warning("skipping unknown mobile device in AutoAway rule")
			continue
		}
		if zone = controller.lookupZone(autoAwayRule.ZoneID, autoAwayRule.ZoneName); zone == nil {
			log.WithFields(log.Fields{
				"zoneID":   autoAwayRule.ZoneID,
				"zoneName": autoAwayRule.ZoneName,
			}).Warning("skipping unknown zone in AutoAway rule")
			continue
		}

		// Add/update the entry in the AutoAwayInfo map
		if entry, ok := controller.AutoAwayInfo[mobileDevice.ID]; ok == false {
			// We don't already have a record. Create it
			controller.AutoAwayInfo[mobileDevice.ID] = AutoAwayInfo{
				MobileDevice:   mobileDevice,
				ZoneID:         zone.ID,
				AutoAwayRule:   autoAwayRule,
				state:          getInitialState(mobileDevice),
				ActivationTime: time.Now().Add(autoAwayRule.WaitTime),
			}
		} else {
			// If we already have it, update it
			entry.MobileDevice = mobileDevice
			controller.AutoAwayInfo[mobileDevice.ID] = entry
		}

	}

	return err
}

// getAutoAwayActions scans the AutoAwayInfo map and returns all required actions, i.e. any zones that
// need to be put in/out of Overlay mode.
func (controller *Controller) getAutoAwayActions() ([]action, error) {
	var (
		err     error
		actions = make([]action, 0)
	)

	for id, autoAwayInfo := range controller.AutoAwayInfo {
		log.WithFields(log.Fields{
			"mobileDeviceID":   autoAwayInfo.MobileDevice.ID,
			"mobileDeviceName": autoAwayInfo.MobileDevice.Name,
			"state":            autoAwayInfo.state,
			"new_home":         autoAwayInfo.MobileDevice.Location.AtHome,
			"activation_time":  autoAwayInfo.ActivationTime,
		}).Debug("autoAwayInfo")

		// if the mobile phone is now home but was away
		if autoAwayInfo.cameHome() {
			// mark the phone at home
			autoAwayInfo.state = autoAwayStateHome
			controller.AutoAwayInfo[id] = autoAwayInfo
			// add action to disable the overlay
			actions = append(actions, action{
				Overlay: false,
				ZoneID:  autoAwayInfo.ZoneID,
			})
			log.WithFields(log.Fields{
				"MobileDeviceID": id,
				"ZoneID":         autoAwayInfo.ZoneID,
			}).Info("User returned home. Removing overlay")
			// notify via slack if needed
			mobileDevice, _ := controller.MobileDevices[id]
			err = controller.notify(
				fmt.Sprintf("%s is home. switching off manual control in zone %s",
					mobileDevice.Name,
					controller.zoneName(autoAwayInfo.ZoneID),
				),
			)
		} else
		// if the mobile phone is away, mark it as such and set the activation timer
		if autoAwayInfo.leftHome() {
			autoAwayInfo.state = autoAwayStateAway
			autoAwayInfo.ActivationTime = time.Now().Add(autoAwayInfo.AutoAwayRule.WaitTime)
			controller.AutoAwayInfo[id] = autoAwayInfo
		} else
		// if the mobile phone was already away, check the activation timer
		if autoAwayInfo.shouldReport() {

			autoAwayInfo.state = autoAwayStateReported
			controller.AutoAwayInfo[id] = autoAwayInfo
			// add action to set the overlay
			actions = append(actions, action{
				Overlay:           true,
				ZoneID:            autoAwayInfo.ZoneID,
				TargetTemperature: autoAwayInfo.AutoAwayRule.TargetTemperature,
			})
			log.WithFields(log.Fields{
				"MobileDeviceID":    id,
				"ZoneID":            autoAwayInfo.ZoneID,
				"TargetTemperature": autoAwayInfo.AutoAwayRule.TargetTemperature,
			}).Info("User left. Setting overlay")
			// notify via slack if needed
			mobileDevice, _ := controller.MobileDevices[id]
			err = controller.notify(
				fmt.Sprintf("%s is away. activating manual control in zone %s",
					mobileDevice.Name,
					controller.zoneName(autoAwayInfo.ZoneID),
				),
			)
		}
	}
	return actions, err
}
