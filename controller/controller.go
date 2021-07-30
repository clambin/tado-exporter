package controller

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
)

// Controller object for tado-controller.
type Controller struct {
	API         tado.API
	ZoneManager *zonemanager.Manager
	TadoBot     *slackbot.SlackBot
}

// New creates a new Controller object
func New(API tado.API, cfg *configuration.ControllerConfiguration, tadoBot *slackbot.SlackBot) (controller *Controller, err error) {
	var postChannel slackbot.PostChannel
	if tadoBot != nil {
		postChannel = tadoBot.PostChannel
	}

	var mgr *zonemanager.Manager
	mgr, err = zonemanager.New(API, cfg.ZoneConfig, postChannel)
	if err == nil {
		if tadoBot != nil {
			tadoBot.RegisterCallback("rules", mgr.ReportTasks)
		}
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

	go func() {
		_ = controller.ZoneManager.Run(ctx)
	}()

	<-ctx.Done()

	log.Info("controller stopped")
}
