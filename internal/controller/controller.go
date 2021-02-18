package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/internal/tadoproxy"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// Controller object for tado-controller.
type Controller struct {
	//tado.API
	Configuration *configuration.ControllerConfiguration
	TadoBot       *tadobot.TadoBot

	proxy        tadoproxy.Proxy
	AutoAwayInfo map[int]AutoAwayInfo
	Overlays     map[int]time.Time
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller, err error) {
	controller = &Controller{
		Configuration: cfg,
		proxy: tadoproxy.Proxy{
			API: &tado.APIClient{
				HTTPClient:   &http.Client{},
				Username:     tadoUsername,
				Password:     tadoPassword,
				ClientSecret: tadoClientSecret,
			},
		},
	}

	if cfg.SlackbotToken != "" {
		callbacks := map[string]tadobot.CallbackFunc{
			"rooms":        controller.doRooms,
			"users":        controller.doUsers,
			"rules":        controller.doRules,
			"autoaway":     controller.doRulesAutoAway,
			"limitoverlay": controller.doRulesLimitOverlay,
		}
		if controller.TadoBot, err = tadobot.Create(
			cfg.SlackbotToken,
			tadoUsername,
			tadoPassword,
			tadoClientSecret,
			callbacks,
		); err == nil {
			go func() {
				controller.TadoBot.Run()
			}()
		} else {
			log.WithField("err", "failed to start TadoBot")
		}
	}

	return
}

// Run executes all controller rules
func (controller *Controller) Run() error {
	err := controller.proxy.Refresh()

	if err == nil {
		err = controller.runAutoAway()
	}

	if err == nil {
		err = controller.runOverlayLimit()
	}

	log.WithField("err", err).Debug("Run")

	return err
}

// lookupMobileDevice returns the mobile device matching the mobileDeviceID or mobileDeviceName from the list of mobile devices
func (controller *Controller) lookupMobileDevice(mobileDeviceID int, mobileDeviceName string) *tado.MobileDevice {
	var (
		ok           bool
		mobileDevice *tado.MobileDevice
	)

	if mobileDevice, ok = controller.proxy.MobileDevice[mobileDeviceID]; ok == false {
		for _, mobileDevice = range controller.proxy.MobileDevice {
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

	if zone, ok = controller.proxy.Zone[zoneID]; ok == false {
		for _, zone = range controller.proxy.Zone {
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
	if controller.TadoBot != nil {
		err = controller.TadoBot.SendMessage("", message)
	}
	return err
}

func (controller *Controller) zoneName(zoneID int) string {
	if zone, ok := controller.proxy.Zone[zoneID]; ok {
		return zone.Name
	}
	return "unknown"
}
