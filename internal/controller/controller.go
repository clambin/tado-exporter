package controller

import (
	"context"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/tado-exporter/internal/controller/zone"
	"github.com/clambin/tado-exporter/internal/controller/zone/notifier"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

// Controller object for tado-controller
type Controller struct {
	ZoneManagers zone.Controllers
	tasks        taskmanager.Manager
	logger       *slog.Logger
}

// New creates a new Controller object
func New(api rules.TadoSetter, cfg []rules.ZoneConfig, tadoBot notifier.SlackSender, p poller.Poller, logger *slog.Logger) *Controller {
	c := Controller{logger: logger}

	for _, zoneCfg := range cfg {
		z := zone.New(api, p, tadoBot, zoneCfg, logger.With("zone", zoneCfg.Zone))
		c.ZoneManagers = append(c.ZoneManagers, z)
		_ = c.tasks.Add(z)
	}

	return &c
}

// Run the controller
func (c *Controller) Run(ctx context.Context) error {
	c.logger.Debug("started")
	defer c.logger.Debug("stopped")
	return c.tasks.Run(ctx)
}
