package controller

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

// AutoAwayInfo contains the user we are tracking, and what zone to set to which temperature
// when ActivationTime occurs
type AutoAwayInfo struct {
	MobileDevice   *tado.MobileDevice
	Home           bool
	ActivationTime time.Time
	ZoneID         int
	AutoAwayRule   *configuration.AutoAwayRule
}

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
func (controller *Controller) updateAutoAwayInfo() error {
	var err error

	// If the map doesn't exist, create it
	if controller.AutoAwayInfo == nil {
		controller.AutoAwayInfo = make(map[int]AutoAwayInfo)
	}

	// for each autoAway setting, add/update a record for the mobileDevice
	for _, autoAway := range *controller.Configuration.AutoAwayRules {
		var (
			mobileDevice *tado.MobileDevice
			zone         *tado.Zone
		)
		// Rules file can contain either mobileDevice/zone ID or Name. Retrieve the ID for each of these
		// and discard any that aren't valid
		if mobileDevice = controller.lookupMobileDevice(autoAway.MobileDeviceID, autoAway.MobileDeviceName); mobileDevice == nil {
			log.WithFields(log.Fields{
				"deviceID":   autoAway.MobileDeviceID,
				"deviceName": autoAway.MobileDeviceName,
			}).Warning("skipping unknown mobile device in AutoAway rule")
			continue
		}
		if zone = controller.lookupZone(autoAway.ZoneID, autoAway.ZoneName); zone == nil {
			log.WithFields(log.Fields{
				"zoneID":   autoAway.ZoneID,
				"zoneName": autoAway.ZoneName,
			}).Warning("skipping unknown zone in AutoAway rule")
			continue
		}

		// Add/update the entry in the AutoAwayInfo map
		if entry, ok := controller.AutoAwayInfo[mobileDevice.ID]; ok == false {
			// We don't already have a record. Create it
			controller.AutoAwayInfo[mobileDevice.ID] = AutoAwayInfo{
				MobileDevice:   mobileDevice,
				ZoneID:         zone.ID,
				AutoAwayRule:   autoAway,
				Home:           mobileDevice.Location.AtHome,
				ActivationTime: time.Now().Add(autoAway.WaitTime),
			}
		} else {
			// If we already have it, update it
			entry.MobileDevice = mobileDevice
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
			"new_home":         autoAwayInfo.MobileDevice.Location.AtHome,
			"old_home":         autoAwayInfo.Home,
		}).Debug("autoAwayInfo")

		// if the mobile phone is now home but was away
		if autoAwayInfo.MobileDevice.Location.AtHome && !autoAwayInfo.Home {
			// mark the phone at home
			autoAwayInfo.Home = true
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
		// if the mobile phone is away
		if !autoAwayInfo.MobileDevice.Location.AtHome {
			// if the phone was home, mark the phone away & record the time
			if autoAwayInfo.Home {
				autoAwayInfo.Home = false
				autoAwayInfo.ActivationTime = time.Now().Add(autoAwayInfo.AutoAwayRule.WaitTime)
				controller.AutoAwayInfo[id] = autoAwayInfo
			}
			// if the phone's been away for the required time
			if time.Now().After(autoAwayInfo.ActivationTime) {
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
						controller.zoneName(autoAwayInfo.AutoAwayRule.ZoneID),
					),
				)
			}
		}
	}
	return actions, err
}
