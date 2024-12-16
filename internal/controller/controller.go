package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

type TadoClient interface {
	SetPresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, body tado.SetPresenceLockJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error)
	DeletePresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.DeletePresenceLockResponse, error)
	SetZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, body tado.SetZoneOverlayJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error)
	DeleteZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, reqEditors ...tado.RequestEditorFn) (*tado.DeleteZoneOverlayResponse, error)
}

type Publisher[T any] interface {
	Subscribe() <-chan T
	Unsubscribe(<-chan T)
}

// A Controller creates and runs the required home and zone evaluators for a Configuration.
type Controller struct {
	logger      *slog.Logger
	controllers []*groupEvaluator
}

type Configuration struct {
	Zones map[string][]rules.RuleConfiguration `yaml:"zones"`
	Home  []rules.RuleConfiguration            `yaml:"home"`
}

// New creates a new Controller, with the home & zone controllers required by the Configuration.
func New(cfg Configuration, p Publisher[poller.Update], c TadoClient, n notifier.Notifier, l *slog.Logger) (*Controller, error) {
	m := Controller{logger: l}
	if len(cfg.Home) > 0 {
		homeRules, err := rules.LoadHomeRules(cfg.Home)
		if err != nil {
			return nil, fmt.Errorf("could not load home rules: %w", err)
		}
		m.controllers = append(
			m.controllers,
			newGroupEvaluator(homeRules, p, c, n, l.With(slog.String("module", "home controller"))),
		)
	}
	for zoneName, zoneCfg := range cfg.Zones {
		if len(zoneCfg) == 0 {
			continue
		}
		zoneRules, err := rules.LoadZoneRules(zoneName, zoneCfg)
		if err != nil {
			return nil, fmt.Errorf("could not load rules for zone %q: %w", zoneName, err)
		}
		m.controllers = append(
			m.controllers,
			newGroupEvaluator(zoneRules, p, c, n, l.With(slog.String("module", "zone controller"), slog.String("zone", zoneName))),
		)
	}
	return &m, nil
}

// Run starts all controllers and waits for them to terminate.
func (m *Controller) Run(ctx context.Context) error {
	m.logger.Debug("controller starting")
	defer m.logger.Debug("controller stopping")

	var g errgroup.Group
	for _, controller := range m.controllers {
		g.Go(func() error { return controller.Run(ctx) })
	}
	return g.Wait()
}

func (m *Controller) ReportTasks() []string {
	tasks := make([]string, 0, len(m.controllers))
	for _, c := range m.controllers {
		if task := c.ReportTask(); task != "" {
			tasks = append(tasks, task)
		}
	}
	return tasks
}
