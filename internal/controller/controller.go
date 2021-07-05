package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/slackbot"
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
	stop    chan struct{}
	ticker  *time.Ticker
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller, err error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("controller not enabled")
	}
	var tadoBot *slackbot.SlackBot
	var postChannel slackbot.PostChannel
	if cfg.TadoBot.Enabled {
		if tadoBot, err = slackbot.Create("tado "+version.BuildVersion, cfg.TadoBot.Token.Value, nil); err == nil {
			go func() {
				_ = tadoBot.Run()
			}()
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
func (controller *Controller) Run(ctx context.Context) {
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
			}
		}
	}
	close(controller.stop)
}
