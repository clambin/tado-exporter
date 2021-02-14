package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/containrrr/shoutrrr"
	log "github.com/sirupsen/logrus"
	"time"
)

// Controller object for tado-controller.
type Controller struct {
	tado.API
	Configuration *configuration.ControllerConfiguration
	TadoBot       *tadobot.TadoBot

	Zones         map[int]*tado.Zone
	MobileDevices map[int]*tado.MobileDevice
	AutoAwayInfo  map[int]AutoAwayInfo
	Overlays      map[int]time.Time
}

// Configuration options for tado-exporter
type Configuration struct {
	Username     string
	Password     string
	ClientSecret string
	Interval     time.Duration
	NotifyURL    string
	Port         int
	Debug        bool
}

// Run executes all controller rules
func (controller *Controller) Run() error {
	err := controller.updateTadoConfig()

	if err == nil {
		err = controller.runAutoAway()
	}

	if err == nil {
		err = controller.runOverlayLimit()
	}

	log.WithField("err", err).Debug("Run")

	return err
}

func (controller *Controller) updateTadoConfig() error {
	var (
		err           error
		zones         []*tado.Zone
		mobileDevices []*tado.MobileDevice
	)

	if zones, err = controller.GetZones(); err == nil {
		zoneMap := make(map[int]*tado.Zone)
		for _, zone := range zones {
			zoneMap[zone.ID] = zone
		}
		controller.Zones = zoneMap

		if mobileDevices, err = controller.GetMobileDevices(); err == nil {
			mobileDeviceMap := make(map[int]*tado.MobileDevice)
			for _, mobileDevice := range mobileDevices {
				mobileDeviceMap[mobileDevice.ID] = mobileDevice
			}
			controller.MobileDevices = mobileDeviceMap
		}
	}

	log.WithFields(log.Fields{
		"err":           err,
		"zones":         len(controller.Zones),
		"mobileDevices": len(controller.MobileDevices),
	}).Debug("updateTadoConfig")

	return err
}

// lookupMobileDevice returns the mobile device matching the mobileDeviceID or mobileDeviceName from the list of mobile devices
func (controller *Controller) lookupMobileDevice(mobileDeviceID int, mobileDeviceName string) *tado.MobileDevice {
	var (
		ok           bool
		mobileDevice *tado.MobileDevice
	)

	if mobileDevice, ok = controller.MobileDevices[mobileDeviceID]; ok == false {
		for _, mobileDevice = range controller.MobileDevices {
			if mobileDeviceName == mobileDevice.Name {
				ok = true
				break
			}
		}
	}

	if ok == false {
		return nil
	}
	return mobileDevice
}

// lookupZone returns the zone matching zoneID or zoneName from the list of zones
func (controller *Controller) lookupZone(zoneID int, zoneName string) *tado.Zone {
	var (
		ok   bool
		zone *tado.Zone
	)

	if zone, ok = controller.Zones[zoneID]; ok == false {
		for _, zone = range controller.Zones {
			if zoneName == zone.Name {
				ok = true
				break
			}
		}
	}

	if ok == false {
		return nil
	}
	return zone
}

// notify sends a message to slack
func (controller *Controller) notify(message string) error {
	var err error
	if controller.Configuration.NotifyURL != "" {
		err = shoutrrr.Send(controller.Configuration.NotifyURL, message)
	}
	if controller.TadoBot != nil {
		err = controller.TadoBot.PostMessage(message)
	}
	return err
}

func (controller *Controller) zoneName(zoneID int) string {
	if zone, ok := controller.Zones[zoneID]; ok {
		return zone.Name
	}
	return "unknown"
}
