package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/poller"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/version"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// Controller object for tado-controller.
type Controller struct {
	API     tado.API
	poller  *poller.Poller
	mgr     *zonemanager.Manager
	tadoBot *slackbot.SlackBot
	ticker  *time.Ticker
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller, err error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("controller not enabled")
	}

	var (
		tadoBot     *slackbot.SlackBot
		postChannel slackbot.PostChannel
		mgr         *zonemanager.Manager
	)

	if cfg.TadoBot.Enabled {
		tadoBot, err = slackbot.Create("tado "+version.BuildVersion, cfg.TadoBot.Token.Value, nil)

		if err != nil {
			return nil, fmt.Errorf("failed to start TadoBot: %s", err.Error())
		}

		go func() {
			// TODO: start this in controller.Run() with context?
			_ = tadoBot.Run()
		}()
		postChannel = tadoBot.PostChannel
	}

	API := &tado.APIClient{
		HTTPClient:   &http.Client{},
		Username:     tadoUsername,
		Password:     tadoPassword,
		ClientSecret: tadoClientSecret,
	}

	pollr := poller.New(API)
	mgr, err = zonemanager.New(API, cfg.ZoneConfig, postChannel)

	if err != nil {
		return nil, fmt.Errorf("failed to create zone manager: %s", err.Error())
	}
	if tadoBot != nil {
		tadoBot.RegisterCallback("rules", mgr.ReportTasks)
	}

	return NewWith(API, pollr, mgr, tadoBot, cfg.Interval)
}

// NewWith creates a controller with pre-existing components.  Used for unit-testing
func NewWith(API tado.API, pollr *poller.Poller, mgr *zonemanager.Manager, tadoBot *slackbot.SlackBot, interval time.Duration) (controller *Controller, err error) {
	controller = &Controller{
		API:     API,
		poller:  pollr,
		mgr:     mgr,
		tadoBot: tadoBot,
		ticker:  time.NewTicker(interval),
	}
	return
}

// Run the controller
func (controller *Controller) Run(ctx context.Context) {
	log.Info("controller started")
	go func() {
		_ = controller.mgr.Run(ctx)
	}()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-controller.ticker.C:
			if updates, err := controller.poller.Update(ctx); err == nil {
				controller.mgr.Update <- updates
			} else {
				log.WithError(err).Warning("failed to get Tado statistics")
			}
		}
	}
	log.Info("controller stopped")
}
