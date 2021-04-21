package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
	//"github.com/clambin/tado-exporter/internal/controller/commands"
)

// Controller object for tado-controller.
type Controller struct {
	API     tado.API
	poller  *poller.Poller
	mgr     *zonemanager.Manager
	tadoBot *slackbot.SlackBot
	stop    chan struct{}
	ticker  *time.Ticker
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller, err error) {
	if cfg != nil && cfg.Enabled {
		var tadoBot *slackbot.SlackBot
		var postChannel slackbot.PostChannel
		if cfg.TadoBot.Enabled {
			if tadoBot, err = slackbot.Create("tado "+version.BuildVersion, cfg.TadoBot.Token.Value, nil); err == nil {
				go tadoBot.Run()
				postChannel = tadoBot.PostChannel
			} else {
				log.WithField("err", "failed to start TadoBot")
				tadoBot = nil
			}
		}

		API := &tado.APIClient{
			HTTPClient:   &http.Client{},
			Username:     tadoUsername,
			Password:     tadoPassword,
			ClientSecret: tadoClientSecret,
		}

		pollr := poller.New(API)

		var mgr *zonemanager.Manager
		mgr, err = zonemanager.New(API, *cfg.ZoneConfig, postChannel)

		if err == nil {
			if tadoBot != nil {
				tadoBot.RegisterCallback("rules", mgr.ReportTasks)
			}
			controller, err = NewWith(API, pollr, mgr, tadoBot, cfg.Interval)
		}
	}
	return
}

// NewWith creates a controller with pre-existing components.  Used for unit-testing
func NewWith(API tado.API, pollr *poller.Poller, mgr *zonemanager.Manager, tadoBot *slackbot.SlackBot, interval time.Duration) (controller *Controller, err error) {
	controller = &Controller{
		API:     API,
		poller:  pollr,
		mgr:     mgr,
		stop:    make(chan struct{}),
		tadoBot: tadoBot,
		ticker:  time.NewTicker(interval),
	}
	return
}

// Run the controller
func (controller *Controller) Run() {
	go controller.mgr.Run()

loop:
	for {
		select {
		case <-controller.ticker.C:
			if updates, err := controller.poller.Update(); err == nil {
				controller.mgr.Update <- updates
			}
		case <-controller.stop:
			controller.mgr.Cancel <- struct{}{}
			break loop
		}
	}
	close(controller.stop)
}

// Stop the controller
func (controller *Controller) Stop() {
	controller.stop <- struct{}{}
}
