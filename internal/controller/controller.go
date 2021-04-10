package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"

	// "github.com/slack-go/slack"
	//"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/pkg/slackbot"
)

// Controller object for tado-controller.
type Controller struct {
	API       tado.API
	poller    *poller.Poller
	mgr       *zonemanager.Manager
	scheduler scheduler.API
	tadoBot   *slackbot.SlackBot
	stop      chan struct{}
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

		pollr := poller.New(API, cfg.Interval)
		schedulr := scheduler.New(API, postChannel)

		if tadoBot != nil {
			tadoBot.RegisterCallback("rules", schedulr.ReportTasks)
		}

		var mgr *zonemanager.Manager
		mgr, err = zonemanager.New(API, *cfg.ZoneConfig, pollr.Update, schedulr)

		if err == nil {
			controller, err = NewWith(API, pollr, mgr, schedulr, tadoBot)
		}
	}
	return
}

// NewWith creates a controller with pre-existing components.  Used for unit-testing
func NewWith(API tado.API, pollr *poller.Poller, mgr *zonemanager.Manager, schedulr scheduler.API, tadoBot *slackbot.SlackBot) (controller *Controller, err error) {
	controller = &Controller{
		API:       API,
		poller:    pollr,
		mgr:       mgr,
		scheduler: schedulr,
		stop:      make(chan struct{}),
		tadoBot:   tadoBot,
	}
	return
}

// Run the controller
func (controller *Controller) Run() {
	go controller.scheduler.Run()
	go controller.mgr.Run()
	go controller.poller.Run()

loop:
	for {
		select {
		case <-controller.stop:
			controller.poller.Cancel <- struct{}{}
			controller.mgr.Cancel <- struct{}{}
			controller.scheduler.Stop()
			break loop
		}
	}
	close(controller.stop)
}

// Stop the controller
func (controller *Controller) Stop() {
	controller.stop <- struct{}{}
}
