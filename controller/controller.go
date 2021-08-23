package controller

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/cache"
	"github.com/clambin/tado-exporter/controller/scheduler"
	"github.com/clambin/tado-exporter/controller/statemanager"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
)

// Controller object for tado-controller
type Controller struct {
	tado.API
	Updates chan *poller.Update

	scheduler    scheduler.Scheduler
	stateManager statemanager.Manager
	cache        cache.Cache
	bot          slackbot.SlackBot
	poller       poller.Poller
}

// New creates a new Controller object
func New(API tado.API, cfg *configuration.ControllerConfiguration, tadoBot slackbot.SlackBot, p poller.Poller) (controller *Controller, err error) {
	controller = &Controller{
		API:     API,
		Updates: make(chan *poller.Update),
		bot:     tadoBot,
		poller:  p,
	}
	controller.stateManager = statemanager.Manager{
		ZoneConfig: cfg.ZoneConfig,
		Cache:      &controller.cache,
	}

	if tadoBot != nil {
		tadoBot.RegisterCallback("rules", controller.ReportRules)
		tadoBot.RegisterCallback("rooms", controller.ReportRooms)
		tadoBot.RegisterCallback("set", controller.SetRoom)
		tadoBot.RegisterCallback("refresh", controller.DoRefresh)
	}

	return controller, err
}

// Run the controller
func (controller *Controller) Run(ctx context.Context) {
	log.Info("controller started")

	go controller.scheduler.Run(ctx)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case update := <-controller.Updates:
			controller.Update(ctx, update)
		}
	}

	log.Info("controller stopped")
}

func (controller *Controller) Update(ctx context.Context, update *poller.Update) {
	controller.cache.Update(update)
	for zoneID, zoneInfo := range update.ZoneInfo {

		state := zoneInfo.GetState()
		targetState, when, reason, err := controller.stateManager.GetNextState(zoneID, update)

		if err != nil {
			log.WithError(err).Warning("failed to get zone state")
			continue
		}

		if targetState != state {
			// log.WithFields(log.Fields{"old": state, "new": targetState, "id": zoneID}).Debug("new zone state determined")

			// schedule the new state
			controller.scheduleZoneStateChange(ctx, zoneID, targetState, when, reason)
		} else {
			// already at target state. cancel any outstanding tasks
			controller.cancelZoneStateChange(zoneID, reason)
		}
	}
	return
}

func (controller *Controller) refresh() {
	if controller.poller != nil {
		controller.poller.Refresh()
	}
}
