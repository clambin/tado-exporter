package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
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

		controller = &Controller{
			proxy:  proxy,
			mgr:    zonemanager.New(*cfg.ZoneConfig, proxy),
			ticker: time.NewTicker(cfg.Interval),
			stop:   make(chan struct{}),
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
func (controller *Controller) Run() {
loop:
	for {
		select {
		case <-controller.ticker.C:
			if updates := controller.mgr.Update(); len(updates) > 0 {
				controller.proxy.SetZones <- updates

				for id, state := range updates {
					log.WithFields(log.Fields{
						"zone":  controller.mgr.AllZones[id],
						"state": state.String(),
					}).Info("setting zone state")

					// TODO: send a message to slack
				}
			}
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
