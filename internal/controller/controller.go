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
	tasks        taskmanager.Manager
	logger       *slog.Logger
}

type TadoSetter interface {
	rules.TadoSetter
	commands.TadoSetter
}

// New creates a new Controller object
func New(api TadoSetter, cfg []rules.ZoneConfig, tadoBot slackbot.SlackBot, p poller.Poller, logger *slog.Logger) *Controller {
	c := Controller{logger: logger}

	for _, zoneCfg := range cfg {
		z := zone.New(api, p, tadoBot, zoneCfg, logger.With("zone", zoneCfg.Zone))
		c.zoneManagers = append(c.zoneManagers, z)
		_ = c.tasks.Add(z)
	}

	if tadoBot != nil {
		c.Commands = commands.New(api, tadoBot, p, c.zoneManagers, logger.With("component", "commands"))
		_ = c.tasks.Add(c.Commands)
	}

	return &c
}

// Run the controller
func (c *Controller) Run(ctx context.Context) error {
	c.logger.Debug("started")
	defer c.logger.Debug("stopped")
	return c.tasks.Run(ctx)
}
