package controller

import (
	"context"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/slackbot"
	"github.com/clambin/tado-exporter/internal/controller/zone"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

// Controller object for tado-controller
type Controller struct {
	zoneManagers zone.Controllers
	Commands     *commands.Executor
}

type TadoSetter interface {
	rules.TadoSetter
	commands.TadoSetter
}

// New creates a new Controller object
func New(api TadoSetter, cfg []rules.ZoneConfig, tadoBot slackbot.SlackBot, p poller.Poller) *Controller {
	var c Controller

	for _, zoneCfg := range cfg {
		c.zoneManagers = append(c.zoneManagers, zone.New(api, p, tadoBot, zoneCfg))
	}

	if tadoBot != nil {
		c.Commands = commands.New(api, tadoBot, p, c.zoneManagers)
	}

	return &c
}

// Run the controller
func (c *Controller) Run(ctx context.Context) error {
	slog.Info("controller started")
	defer slog.Info("controller stopped")

	var mgrs []taskmanager.Task
	for _, zoneManager := range c.zoneManagers {
		mgrs = append(mgrs, zoneManager)
	}
	tm := taskmanager.New(mgrs...)
	if c.Commands != nil {
		_ = tm.Add(c.Commands)
	}

	return tm.Run(ctx)
}
