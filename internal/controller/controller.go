package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/home"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/zone"
	"github.com/clambin/tado-exporter/internal/poller"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

// Controller for tado-controller
type Controller struct {
	tasks  []task
	logger *slog.Logger
}

type task interface {
	Run(ctx context.Context) error
	ReportTask() (string, bool)
}

// New creates a new Controller object
func New(api action.TadoClient, cfg configuration.Configuration, s notifier.SlackSender, p poller.Poller, logger *slog.Logger) *Controller {
	c := Controller{logger: logger}

	if cfg.Home.AutoAway.IsActive() {
		h := home.New(api, p, s, cfg.Home, logger.With("type", "home"))
		c.tasks = append(c.tasks, h)
	}

	for _, zoneCfg := range cfg.Zones {
		z := zone.New(api, p, s, zoneCfg, logger.With("type", "zone", "zone", zoneCfg.Name))
		c.tasks = append(c.tasks, z)
	}

	return &c
}

// Run the controller
func (c *Controller) Run(ctx context.Context) error {
	c.logger.Debug("started")
	defer c.logger.Debug("stopped")

	var g errgroup.Group
	for _, t := range c.tasks {
		g.Go(func() error { return t.Run(ctx) })
	}

	return g.Wait()
}

func (c *Controller) ReportTasks() []string {
	reports := make([]string, 0, len(c.tasks))
	for _, t := range c.tasks {
		if report, ok := t.ReportTask(); ok {
			reports = append(reports, report)
		}
	}
	return reports
}
