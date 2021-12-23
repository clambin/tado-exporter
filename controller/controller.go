package controller

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/cache"
	"github.com/clambin/tado-exporter/controller/processor"
	"github.com/clambin/tado-exporter/controller/setter"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
	"time"
)

// Controller object for tado-controller
type Controller struct {
	tado.API
	Updates   chan *poller.Update
	processor processor.Processor
	Setter    setter.ZoneSetter
	cache     cache.Cache
	poller    poller.Poller
}

// New creates a new Controller object
func New(API tado.API, cfg *configuration.ControllerConfiguration, tadoBot slackbot.SlackBot, p poller.Poller) (controller *Controller) {
	controller = &Controller{
		API:       API,
		Updates:   make(chan *poller.Update),
		processor: processor.New(cfg.ZoneConfig),
		Setter:    setter.New(API, tadoBot),
		poller:    p,
	}

	if tadoBot != nil {
		tadoBot.RegisterCallback("rules", controller.ReportRules)
		tadoBot.RegisterCallback("rooms", controller.ReportRooms)
		tadoBot.RegisterCallback("set", controller.SetRoom)
		tadoBot.RegisterCallback("refresh", controller.DoRefresh)
		tadoBot.RegisterCallback("users", controller.ReportUsers)
	}

	return controller
}

// Run the controller
func (controller *Controller) Run(ctx context.Context, interval time.Duration) {
	log.Info("controller started")

	go controller.Setter.Run(ctx, interval)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case update := <-controller.Updates:
			controller.Update(update)
		}
	}

	log.Info("controller stopped")
}

// Update takes the poller's update, determines the next state for each zone and queues that state with the Setter
func (controller *Controller) Update(update *poller.Update) {
	controller.cache.Update(update)
	for zoneID, nextState := range controller.processor.Process(update) {
		if nextState != nil {
			controller.Setter.Set(*nextState)
		} else {
			controller.Setter.Clear(zoneID)
		}
	}
	return
}

func (controller *Controller) refresh() {
	if controller.poller != nil {
		controller.poller.Refresh()
	}
}
