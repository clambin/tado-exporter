package controller

import (
	"context"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/tado-exporter/internal/controller/home"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/zone"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

// Controller object for tado-controller
type Controller struct {
	reporters []TaskReporter
	tasks     taskmanager.Manager
	logger    *slog.Logger
}

type TaskReporter interface {
	ReportTask() (string, bool)
}

// New creates a new Controller object
func New(api action.TadoSetter, cfg configuration.Configuration, s notifier.SlackSender, p poller.Poller, logger *slog.Logger) *Controller {
	c := Controller{logger: logger}

	if cfg.Home.AutoAway.IsActive() {
		h := home.New(api, p, s, cfg.Home, logger.With("type", "home"))
		c.reporters = append(c.reporters, h)
		_ = c.tasks.Add(h)
	}

	for _, zoneCfg := range cfg.Zones {
		z := zone.New(api, p, s, zoneCfg, logger.With("type", "zone", "zone", zoneCfg.Name))
		c.reporters = append(c.reporters, z)
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

func (c *Controller) ReportTasks() []string {
	tasks := make([]string, 0, len(c.reporters))
	for _, r := range c.reporters {
		if t, ok := r.ReportTask(); ok {
			tasks = append(tasks, t)
		}
	}
	return tasks
}
