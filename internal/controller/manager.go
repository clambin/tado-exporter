package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

type Publisher[T any] interface {
	Subscribe() chan T
	Unsubscribe(chan T)
}

type Notifier interface {
	Notify(string)
}

type TadoClient interface {
	SetPresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, body tado.SetPresenceLockJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error)
	DeletePresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.DeletePresenceLockResponse, error)
	SetZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, body tado.SetZoneOverlayJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error)
	DeleteZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, reqEditors ...tado.RequestEditorFn) (*tado.DeleteZoneOverlayResponse, error)
}

// A Manager creates and runs the required home & zone controllers for a Configuration.
type Manager struct {
	controllers []*Controller
	logger      *slog.Logger
}

// NewManager creates a new Manager, with the home & zone controllers required by the Configuration.
func NewManager(cfg Configuration, p Publisher[poller.Update], c TadoClient, n Notifier, l *slog.Logger) (*Manager, error) {
	m := Manager{logger: l}
	if len(cfg.HomeRules) > 0 {
		homeRules, err := LoadHomeRules(cfg.HomeRules)
		if err != nil {
			return nil, fmt.Errorf("could not load home rules: %w", err)
		}
		m.controllers = append(
			m.controllers,
			NewController(homeRules, p, c, n, l.With(slog.String("module", "home controller"))),
		)
	}
	for zoneName, zoneCfg := range cfg.ZoneRules {
		if len(zoneCfg) == 0 {
			continue
		}
		zoneRules, err := LoadZoneRules(zoneName, zoneCfg)
		if err != nil {
			return nil, fmt.Errorf("could not load rules for zone %q: %w", zoneName, err)
		}
		m.controllers = append(
			m.controllers,
			NewController(zoneRules, p, c, n, l.With(slog.String("module", "zone controller"), slog.String("zone", zoneName))),
		)
	}
	return &m, nil
}

// Run starts all controllers and waits for them to terminate.
func (m *Manager) Run(ctx context.Context) error {
	m.logger.Debug("controller manager starting")
	defer m.logger.Debug("controller manager stopping")

	var g errgroup.Group
	for _, controller := range m.controllers {
		g.Go(func() error { return controller.Run(ctx) })
	}
	return g.Wait()
}

func (m *Manager) ReportTasks() []string {
	tasks := make([]string, 0, len(m.controllers))
	for _, c := range m.controllers {
		if task := c.ReportTask(); task != "" {
			tasks = append(tasks, task)
		}
	}
	return tasks
}
