package controller

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

type AutoAwayInfo struct {
	MobileDevice      *tado.MobileDevice
	Home              bool
	Since             time.Time
	WaitTime          time.Duration
	ZoneID            int
	TargetTemperature float64
}

type Action struct {
	Overlay           bool
	ZoneID            int
	TargetTemperature float64
}

func (controller *Controller) AutoAwayRun() error {
	if controller.Rules.AutoAway == nil {
		return nil
	}

	var (
		err     error
		actions []Action
	)

	// update mobiles & zones for each autoaway entry
	if err = controller.AutoAwayUpdateInfo(); err == nil {
		// get actions for each autoaway setting
		if actions, err = controller.AutoAwayGetActions(); err == nil {
			for _, action := range actions {
				// execute each action
				if err = controller.AutoAwayDoAction(action); err != nil {
					break
				}
			}
		}
	}

	log.WithField("err", err).Debug("AutoAwayRun")
	return err
}

// AutoAwayUpdateInfo updates the mobile device & zone information for each autoAway rule.
// On exit, the map controller.AutoAwayInfo contains the up to date mobileDevice information
// for any mobile device mentioned in any autoAway rule.
func (controller *Controller) AutoAwayUpdateInfo() error {
	var (
		err           error
		mobileDevices []tado.MobileDevice
		zones         []tado.Zone
	)

	// If the map doesn't exist, create it
	if controller.AutoAwayInfo == nil {
		controller.AutoAwayInfo = make(map[int]AutoAwayInfo)
	}

	// get info we will need
	if mobileDevices, err = controller.GetMobileDevices(); err == nil {
		zones, err = controller.GetZones()
	}

	if err == nil {
		// for each autoaway setting, add/update a record for the mobileDevice
		for _, autoAway := range *controller.Rules.AutoAway {
			var (
				mobileDevice *tado.MobileDevice
				zone         *tado.Zone
			)
			// Validate the configured mobileDevice & zone ID/Name
			if mobileDevice = getMobileDevice(mobileDevices, autoAway.MobileDeviceID, autoAway.MobileDeviceName); mobileDevice == nil {
				log.WithFields(log.Fields{
					"deviceID":   autoAway.MobileDeviceID,
					"deviceName": autoAway.MobileDeviceName,
				}).Warning("skipping unknown mobile device")
				continue
			}
			if zone = getZone(zones, autoAway.ZoneID, autoAway.ZoneName); zone == nil {
				log.WithFields(log.Fields{
					"zoneID":   autoAway.ZoneID,
					"zoneName": autoAway.ZoneName,
				}).Warning("skipping unknown zone")
				continue
			}

			// Add/update the entry in the AutoAwayInfo map
			if entry, ok := controller.AutoAwayInfo[mobileDevice.ID]; ok == false {
				// If we don't already have a record, create it
				controller.AutoAwayInfo[mobileDevice.ID] = AutoAwayInfo{
					MobileDevice:      mobileDevice,
					Home:              mobileDevice.Location.AtHome,
					Since:             time.Now(),
					ZoneID:            zone.ID,
					WaitTime:          autoAway.WaitTime,
					TargetTemperature: autoAway.TargetTemperature,
				}
			} else {
				// If we already have it, update it
				entry.MobileDevice = mobileDevice
			}
		}
	}

	return err
}

// autoAwayGetActions returns a list of needed actions based on the current status of the autoAway mobile devices
func (controller *Controller) AutoAwayGetActions() ([]Action, error) {
	var (
		err     error
		actions = make([]Action, 0)
	)

	for id, autoAway := range controller.AutoAwayInfo {
		log.WithFields(log.Fields{
			"mobileDeviceID":   autoAway.MobileDevice.ID,
			"mobileDeviceName": autoAway.MobileDevice.Name,
			"new_home":         autoAway.MobileDevice.Location.AtHome,
			"old_home":         autoAway.Home,
		}).Debug("autoAwayInfo")

		// if the mobile phone is now home but was away
		if autoAway.MobileDevice.Location.AtHome && !autoAway.Home {
			// mark the phone at home
			autoAway.Home = true
			controller.AutoAwayInfo[id] = autoAway
			// add action to disable the overlay
			actions = append(actions, Action{
				Overlay: false,
				ZoneID:  autoAway.ZoneID,
			})
			log.WithFields(log.Fields{
				"MobileDeviceID": id,
				"ZoneID":         autoAway.ZoneID,
			}).Info("User returned home. Removing overlay")
		} else
		// if the mobile phone is away
		if !autoAway.MobileDevice.Location.AtHome {
			// if the phone was home, mark the phone away & record the time
			if autoAway.Home {
				autoAway.Home = false
				autoAway.Since = time.Now()
				controller.AutoAwayInfo[id] = autoAway
			}
			// if the phone's been away for the required time
			if time.Now().After(autoAway.Since.Add(autoAway.WaitTime)) {
				// add action to set the overlay
				actions = append(actions, Action{
					Overlay:           true,
					ZoneID:            autoAway.ZoneID,
					TargetTemperature: autoAway.TargetTemperature,
				})
				log.WithFields(log.Fields{
					"MobileDeviceID":    id,
					"ZoneID":            autoAway.ZoneID,
					"TargetTemperature": autoAway.TargetTemperature,
				}).Info("User left. Setting overlay")
			}
		}
	}
	return actions, err
}

// AutoAwayDoAction execute an action
func (controller *Controller) AutoAwayDoAction(action Action) error {
	var (
		err      error
		zoneInfo *tado.ZoneInfo
	)

	if action.Overlay == false {
		// Are we currently in overlay?
		if zoneInfo, err = controller.GetZoneInfo(action.ZoneID); err == nil {
			if zoneInfo.Overlay.Type == "MANUAL" {
				// Delete the overlay
				err = controller.DeleteZoneManualTemperature(action.ZoneID)
			}
		}
	} else {
		// Set the overlay
		err = controller.SetZoneManualTemperature(action.ZoneID, action.TargetTemperature)
	}

	return err
}

// getMobileDevice returns the mobile device matching the mobileDeviceID or mobileDeviceName from the list of mobile devices
func getMobileDevice(mobileDevices []tado.MobileDevice, mobileDeviceID int, mobileDeviceName string) *tado.MobileDevice {
	for _, mobileDevice := range mobileDevices {
		if (mobileDeviceName != "" && mobileDeviceName == mobileDevice.Name) ||
			(mobileDeviceID != 0 && mobileDeviceID == mobileDevice.ID) {
			return &mobileDevice
		}
	}

	return nil
}

// getZone returns the zone matching zoneID or zoneName from the list of zones
func getZone(zones []tado.Zone, zoneID int, zoneName string) *tado.Zone {
	for _, zone := range zones {
		if (zoneName != "" && zoneName == zone.Name) ||
			(zoneID != 0 && zoneID == zone.ID) {
			return &zone
		}
	}

	return nil

}
