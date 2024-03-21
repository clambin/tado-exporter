package monitor

import (
	"errors"
	"fmt"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/clambin/go-common/slackbot"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/go-common/taskmanager/httpserver"
	promserver "github.com/clambin/go-common/taskmanager/prometheus"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/bot"
	"github.com/clambin/tado-exporter/internal/collector"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/health"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

var _ prometheus.Collector = &Monitor{}

type Monitor struct {
	*taskmanager.Manager
	collector *collector.Collector
}

func New(cfg *viper.Viper, version string, metrics roundtripper.RoundTripMetrics, logger *slog.Logger) (*Monitor, error) {
	api, err := tadotools.GetInstrumentedTadoClient(
		cfg.GetString("tado.username"),
		cfg.GetString("tado.password"),
		cfg.GetString("tado.clientSecret"),
		metrics,
	)
	if err != nil {
		return nil, fmt.Errorf("tado: %w", err)
	}

	// Do we have zone rules?
	rules, err := maybeLoadRules(filepath.Join(filepath.Dir(cfg.ConfigFileUsed()), "rules.yaml"))
	if err != nil {
		return nil, err
	}

	return makeMonitor(cfg, api, version, rules, logger), nil
}

func maybeLoadRules(path string) (configuration.Configuration, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return configuration.Configuration{}, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	return configuration.Load(f)
}

func makeMonitor(cfg *viper.Viper, api *tado.APIClient, version string, rules configuration.Configuration, l *slog.Logger) *Monitor {
	var monitor Monitor
	monitor.Manager = taskmanager.New(monitor.makeTasks(cfg, api, version, rules, l)...)
	return &monitor
}

func (m *Monitor) makeTasks(cfg *viper.Viper, api *tado.APIClient, version string, rules configuration.Configuration, l *slog.Logger) []taskmanager.Task {
	var tasks []taskmanager.Task

	// Poller
	p := poller.New(api, cfg.GetDuration("poller.interval"), l.With("component", "poller"))
	tasks = append(tasks, p)

	// Collector
	m.collector = &collector.Collector{Poller: p, Logger: l.With("component", "collector")}
	tasks = append(tasks, m.collector)

	// Prometheus Server
	tasks = append(tasks, promserver.New(promserver.WithAddr(cfg.GetString("exporter.addr"))))

	// Health Endpoint
	h := health.New(p, l.With("component", "health"))
	tasks = append(tasks, h)
	r := http.NewServeMux()
	r.Handle("/health", h)
	tasks = append(tasks, httpserver.New(cfg.GetString("health.addr"), r))

	// Controller
	if len(rules.Zones) == 0 {
		return tasks
	}

	// Slackbot
	var s *slackbot.SlackBot
	if token := cfg.GetString("controller.tadoBot.token"); token != "" {
		s = slackbot.New(
			token,
			slackbot.WithName("tadoBot "+version),
			slackbot.WithLogger(l.With(slog.String("component", "slackbot"))),
		)
		tasks = append(tasks, s)
	}

	c := controller.New(api, rules, s, p, l.With("component", "controller"))
	tasks = append(tasks, c)

	if s != nil && cfg.GetBool("controller.tadoBot.enabled") {
		tasks = append(tasks, bot.New(api, s, p, c, l.With(slog.String("component", "tadobot"))))
	}

	return tasks
}

func (m *Monitor) Describe(ch chan<- *prometheus.Desc) {
	m.collector.Describe(ch)
}

func (m *Monitor) Collect(ch chan<- prometheus.Metric) {
	m.collector.Collect(ch)
}
