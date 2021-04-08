package controller

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/slack-go/slack"
	"time"

	//"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/tadoproxy"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	log "github.com/sirupsen/logrus"
)

// Controller object for tado-controller.
type Controller struct {
	proxy   *tadoproxy.Proxy
	mgr     *zonemanager.Manager
	tadoBot *slackbot.SlackBot
	ticker  *time.Ticker
	stop    chan struct{}
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller) {
	if cfg != nil && cfg.Enabled {
		proxy := tadoproxy.New(tadoUsername, tadoPassword, tadoClientSecret)
		go proxy.Run()

		controller = NewWithProxy(proxy, cfg)
	}
	return
}

// NewWithProxy creates a controller with a pre-existing proxy
func NewWithProxy(proxy *tadoproxy.Proxy, cfg *configuration.ControllerConfiguration) (controller *Controller) {
	if cfg != nil && cfg.Enabled {

		controller = &Controller{
			proxy:  proxy,
			mgr:    zonemanager.New(*cfg.ZoneConfig, proxy),
			ticker: time.NewTicker(cfg.Interval),
			stop:   make(chan struct{}),
		}

		if cfg.TadoBot.Enabled {
			callbacks := map[string]slackbot.CommandFunc{
				"rooms": controller.doRooms,
				"users": controller.doUsers,
				//	"rules":        controller.doRules,
				//	// "set":          controller.doSetTemperature,
			}
			var err error
			if controller.tadoBot, err = slackbot.Create("tado "+version.BuildVersion, cfg.TadoBot.Token.Value, callbacks); err == nil {
				go controller.tadoBot.Run()
			} else {
				log.WithField("err", "failed to start TadoBot")
				controller.tadoBot = nil
			}
		}
	}
	return
}

// Run the controller
func (controller *Controller) Run() {
	controller.update()
loop:
	for {
		select {
		case <-controller.ticker.C:
			controller.update()
		case <-controller.stop:
			break loop
		}
	}
	close(controller.stop)
}

// Stop the controller
func (controller *Controller) Stop() {
	controller.stop <- struct{}{}
}

func (controller *Controller) update() {
	updates := controller.mgr.Update()

	if len(updates) > 0 {
		controller.proxy.SetZones <- updates

		if controller.tadoBot != nil {
			controller.tadoBot.PostChannel <- controller.makeAttachments(updates)
		}
	}
}

func (controller *Controller) makeAttachments(updates map[int]model.ZoneState) (attachments []slack.Attachment) {
	for id, state := range updates {

		zoneName := controller.proxy.GetAllZones()[id]

		log.WithFields(log.Fields{
			"zone":  zoneName,
			"state": state.String(),
		}).Info("setting zone state")

		var title string
		switch state.State {
		case model.Off:
			title = "switching off heating in " + zoneName
		case model.Auto:
			title = "switching off manual temperature control in " + zoneName
		case model.Manual:
			title = fmt.Sprintf("setting %s to %.1fº", zoneName, state.Temperature.Celsius)
		default:
			title = "unknown state detected for " + zoneName
		}
		attachments = append(attachments, slack.Attachment{Color: "good", Title: title})
	}

	return
}
