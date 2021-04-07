package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
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
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller) {
	if cfg != nil && cfg.Enabled {

		proxy := tadoproxy.New(tadoUsername, tadoPassword, tadoClientSecret)
		go proxy.Run()

		controller = &Controller{
			proxy: proxy,
			mgr:   zonemanager.New(*cfg.ZoneConfig, cfg.Interval, proxy),
		}

		if cfg.TadoBot.Enabled {
			callbacks := map[string]slackbot.CommandFunc{}
			//	"rooms":        controller.doRooms,
			//	"users":        controller.doUsers,
			//	"rules":        controller.doRules,
			//	// "set":          controller.doSetTemperature,
			//}
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
func (controller *Controller) Run() (err error) {
	controller.mgr.Run()
	return
}

// Stop the controller
func (controller *Controller) Stop() {
	controller.mgr.Stop <- struct{}{}
}
