package controller

import (
	"context"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/tado-exporter/controller/commands"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/poller"
	"log/slog"
)

// Controller object for tado-controller
type Controller struct {
	zoneManagers zonemanager.Managers
	cmds         *commands.Manager
}

type TadoSetter interface {
	rules.TadoSetter
	commands.TadoSetter
}

// New creates a new Controller object
func New(api TadoSetter, cfg []rules.ZoneConfig, tadoBot slackbot.SlackBot, p poller.Poller) *Controller {
	var c Controller

	for _, zoneCfg := range cfg {
		c.zoneManagers = append(c.zoneManagers, zonemanager.New(api, p, tadoBot, zoneCfg))
	}

	if tadoBot != nil {
		c.cmds = commands.New(api, tadoBot, p, c.zoneManagers)
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
	if c.cmds != nil {
		_ = tm.Add(c.cmds)
	}

	return tm.Run(ctx)
}
