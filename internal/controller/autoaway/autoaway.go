package autoaway

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"sort"
	"time"
)

type AutoAway struct {
	Updates    scheduler.UpdateChannel
	RoomSetter chan tadosetter.RoomCommand
	Commands   commands.RequestChannel
	Slack      slackbot.PostChannel
	Rules      []*configuration.AutoAwayRule
	deviceInfo map[int]DeviceInfo
}

// Run waits for updates from the scheduler and evaluates configured autoAway rules
func (autoAway *AutoAway) Run() {
loop:
	for {
		select {
		case tadoData := <-autoAway.Updates:
			if tadoData == nil {
				break loop
			}
			autoAway.updateInfo(tadoData)
			autoAway.setZones()
		case cmd := <-autoAway.Commands:
			switch cmd.Command {
			case commands.Report:
				cmd.Response <- autoAway.Report()
			}
		}
	}
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

// setZones scans the DeviceInfo map and switches on/off any overlays
func (autoAway *AutoAway) setZones() {
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

			// disable the overlay
			autoAway.RoomSetter <- tadosetter.RoomCommand{
				ZoneID: deviceInfo.zone.ID,
				Auto:   true,
			}

			log.WithFields(log.Fields{
				"MobileDeviceID": id,
				"ZoneID":         deviceInfo.zone.ID,
			}).Info("User returned home. Removing overlay")

			// notify via slack if needed
			if autoAway.Slack != nil {
				autoAway.Slack <- []slack.Attachment{{
					Color: "good",
					Title: deviceInfo.mobileDevice.Name + " is home",
					Text:  "resetting " + deviceInfo.zone.Name + " to auto",
				}}
			}
		} else
		// if the mobile phone is away, mark it as such and set the activation timer
		if deviceInfo.leftHome() {
			// mark the phone away & set the activation timer
			deviceInfo.state = autoAwayStateAway
			deviceInfo.activationTime = time.Now().Add(deviceInfo.rule.WaitTime)
			autoAway.deviceInfo[id] = deviceInfo

			// notify via slack if needed
			if autoAway.Slack != nil {
				autoAway.Slack <- []slack.Attachment{{
					Color: "good",
					Title: deviceInfo.mobileDevice.Name + " is away",
					Text:  "will set " + deviceInfo.zone.Name + " to manual in " + deviceInfo.rule.WaitTime.String(),
				}}
			}
		} else
		// if the mobile phone was already away for the configured time
		if deviceInfo.shouldReport() {
			// set the state to reported
			deviceInfo.state = autoAwayStateReported
			autoAway.deviceInfo[id] = deviceInfo

			// set the overlay
			autoAway.RoomSetter <- tadosetter.RoomCommand{
				ZoneID:      deviceInfo.zone.ID,
				Auto:        false,
				Temperature: deviceInfo.rule.TargetTemperature,
			}

			log.WithFields(log.Fields{
				"MobileDeviceID":    id,
				"ZoneID":            deviceInfo.zone.ID,
				"TargetTemperature": deviceInfo.rule.TargetTemperature,
			}).Info("User left. Setting overlay")

			// notify via slack if needed
			if autoAway.Slack != nil {
				autoAway.Slack <- []slack.Attachment{{
					Color: "good",
					Title: deviceInfo.mobileDevice.Name + " is away",
					Text:  "activating manual control in zone " + deviceInfo.zone.Name,
				}}
			}
		}
	}
}

func (autoAway *AutoAway) Report() (output []string) {
	for _, entry := range autoAway.deviceInfo {
		var response string
		switch entry.state {
		case autoAwayStateUndetermined:
			response = "undetermined"
		case autoAwayStateHome:
			response = "home"
		case autoAwayStateAway:
			response = "away. will set " + entry.zone.Name + " to manual in " +
				entry.activationTime.Sub(time.Now()).Round(1*time.Minute).String()
		case autoAwayStateExpired:
			response = "away. " + entry.zone.Name + " will be set to manual"
		case autoAwayStateReported:
			response = "away. " + entry.zone.Name + " is set to manual"
		}
		output = append(output, entry.mobileDevice.Name+" is "+response)
	}
	sort.Strings(output)

	return
}
