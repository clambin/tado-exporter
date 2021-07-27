package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/version"
	log "github.com/sirupsen/logrus"
)

// Controller object for tado-controller.
type Controller struct {
	API         tado.API
	ZoneManager *zonemanager.Manager
	TadoBot     *slackbot.SlackBot
}

// New creates a new Controller object
func New(API tado.API, cfg *configuration.ControllerConfiguration) (controller *Controller, err error) {
	var tadoBot *slackbot.SlackBot
	var postChannel slackbot.PostChannel
	if cfg.TadoBot.Enabled {
		tadoBot, err = slackbot.Create("tado "+version.BuildVersion, cfg.TadoBot.Token.Value, nil)

		if err == nil {
			postChannel = tadoBot.PostChannel
		} else {
			err = fmt.Errorf("failed to start TadoBot: %s", err.Error())
		}
	}

	var mgr *zonemanager.Manager
	if err == nil {
		mgr, err = zonemanager.New(API, cfg.ZoneConfig, postChannel)

		if err != nil {
			err = fmt.Errorf("failed to create zone manager: %s", err.Error())
		}
	}

	if err == nil && tadoBot != nil {
		tadoBot.RegisterCallback("rules", mgr.ReportTasks)
	}

	if err == nil {
		controller = &Controller{
			API:         API,
			ZoneManager: mgr,
			TadoBot:     tadoBot,
		}
	}

	return controller, err
}

// Run the controller
func (controller *Controller) Run(ctx context.Context) {
	log.Info("controller started")
	if controller.TadoBot != nil {
		go func() {
			_ = controller.TadoBot.Run(ctx)
		}()
	}

	go func() {
		_ = controller.ZoneManager.Run(ctx)
	}()

	<-ctx.Done()

	log.Info("controller stopped")
}
