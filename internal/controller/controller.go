package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/autoaway"
	"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/overlaylimit"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// Controller object for tado-controller.
type Controller struct {
	tado.API
	scheduler.Scheduler

	roomSetter *tadosetter.Setter
	tadoBot    *slackbot.SlackBot
	autoAway   *autoaway.AutoAway
	limiter    *overlaylimit.OverlayLimit
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller) {
	controller = &Controller{
		API: &tado.APIClient{
			HTTPClient:   &http.Client{},
			Username:     tadoUsername,
			Password:     tadoPassword,
			ClientSecret: tadoClientSecret,
		},
		roomSetter: &tadosetter.Setter{
			API: &tado.APIClient{
				HTTPClient:   &http.Client{},
				Username:     tadoUsername,
				Password:     tadoPassword,
				ClientSecret: tadoClientSecret,
			},
			ZoneSetter: make(chan tadosetter.RoomCommand),
			Stop:       make(chan bool),
		},
	}
	go controller.roomSetter.Run()

	var slackChannel slackbot.PostChannel
	if cfg != nil && cfg.TadoBot.Enabled {
		callbacks := map[string]slackbot.CommandFunc{
			"rooms":        controller.doRooms,
			"users":        controller.doUsers,
			"rules":        controller.doRules,
			"autoaway":     controller.doRulesAutoAway,
			"limitoverlay": controller.doRulesLimitOverlay,
			"set":          controller.doSetTemperature,
		}
		var err error
		if controller.tadoBot, err = slackbot.Create("tado "+version.BuildVersion, cfg.TadoBot.Token.Value, callbacks); err == nil {
			slackChannel = controller.tadoBot.PostChannel
			go controller.tadoBot.Run()
		} else {
			log.WithField("err", "failed to start TadoBot")
			controller.tadoBot = nil
		}
	}

	if cfg != nil && cfg.AutoAwayRules != nil {
		controller.autoAway = &autoaway.AutoAway{
			Updates:    controller.Register(),
			RoomSetter: controller.roomSetter.ZoneSetter,
			Commands:   make(commands.RequestChannel),
			Slack:      slackChannel,
			Rules:      *cfg.AutoAwayRules,
		}
		go controller.autoAway.Run()
	}

	if cfg != nil && cfg.OverlayLimitRules != nil {
		controller.limiter = &overlaylimit.OverlayLimit{
			Updates:    controller.Register(),
			RoomSetter: controller.roomSetter.ZoneSetter,
			Commands:   make(commands.RequestChannel),
			Slack:      slackChannel,
			Rules:      *cfg.OverlayLimitRules,
		}
		go controller.limiter.Run()
	}

	return
}

// Run runs one update
func (controller *Controller) Run() (err error) {
	var tadoData scheduler.TadoData

	if tadoData, err = controller.refresh(); err == nil {
		controller.Update(tadoData)
	}

	log.WithField("err", err).Debug("Run")

	return
}

// Refresh the Cache
func (controller *Controller) refresh() (tadoData scheduler.TadoData, err error) {
	var (
		zones         []*tado.Zone
		zoneInfo      *tado.ZoneInfo
		mobileDevices []*tado.MobileDevice
	)

	zoneMap := make(map[int]tado.Zone)
	if zones, err = controller.GetZones(); err == nil {
		for _, zone := range zones {
			zoneMap[zone.ID] = *zone
		}
	}
	tadoData.Zone = zoneMap

	if err == nil {
		zoneInfoMap := make(map[int]tado.ZoneInfo)
		for zoneID := range tadoData.Zone {
			if zoneInfo, err = controller.GetZoneInfo(zoneID); err == nil {
				zoneInfoMap[zoneID] = *zoneInfo
			}
		}
		tadoData.ZoneInfo = zoneInfoMap
	}

	if err == nil {
		mobileDeviceMap := make(map[int]tado.MobileDevice)
		if mobileDevices, err = controller.GetMobileDevices(); err == nil {
			for _, mobileDevice := range mobileDevices {
				mobileDeviceMap[mobileDevice.ID] = *mobileDevice
			}
		}
		tadoData.MobileDevice = mobileDeviceMap
	}

	log.WithFields(log.Fields{
		"err":           err,
		"zones":         len(tadoData.Zone),
		"zoneInfos":     len(tadoData.ZoneInfo),
		"mobileDevices": len(tadoData.MobileDevice),
	}).Debug("updateTadoConfig")

	return
}
