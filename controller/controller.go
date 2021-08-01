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

// Controller object for tado-controller.
type Controller struct {
	tado.API
	Update       chan *poller.Update
	Report       chan int
	PostChannel  slackbot.PostChannel
	scheduler    *scheduler.Scheduler
	stateManager *statemanager.Manager
	cache        *cache.Cache
}

// New creates a new Controller object
func New(API tado.API, cfg *configuration.ControllerConfiguration, tadoBot *slackbot.SlackBot) (controller *Controller, err error) {
	var postChannel slackbot.PostChannel
	if tadoBot != nil {
		postChannel = tadoBot.PostChannel
	}

	tadoCache := cache.New()
	var stateManager *statemanager.Manager
	stateManager, err = statemanager.New(cfg.ZoneConfig, tadoCache)

	if err == nil {
		controller = &Controller{
			API:          API,
			Update:       make(chan *poller.Update),
			Report:       make(chan int),
			scheduler:    scheduler.New(),
			stateManager: stateManager,
			PostChannel:  postChannel,
			cache:        tadoCache,
		}

		if tadoBot != nil {
			tadoBot.RegisterCallback("rules", controller.ReportRules)
			tadoBot.RegisterCallback("rooms", controller.ReportRooms)
		}
	}

	return controller, err
}

// Run the controller
func (controller *Controller) Run(ctx context.Context) {
	log.Info("controller started")

	go controller.scheduler.Run(ctx)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case update := <-controller.Update:
			controller.cache.Update(update)
			controller.update(ctx, update)
		case report := <-controller.Report:
			switch report {
			case reportRules:
				controller.reportRules()
			case reportRooms:
				controller.reportRooms()
			}
		}
	}

	log.Info("controller stopped")
}

func (controller *Controller) update(ctx context.Context, update *poller.Update) {
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
