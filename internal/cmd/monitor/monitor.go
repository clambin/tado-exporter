package monitor

import (
	"errors"
	"fmt"
	"github.com/clambin/go-common/slackbot"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/go-common/taskmanager/httpserver"
	promserver "github.com/clambin/go-common/taskmanager/prometheus"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/collector"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/bot"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/health"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/tadotools"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func New(cfg *viper.Viper, version string, registry prometheus.Registerer, logger *slog.Logger) (*taskmanager.Manager, error) {
	api, err := tadotools.GetInstrumentedTadoClient(
		cfg.GetString("tado.username"),
		cfg.GetString("tado.password"),
		cfg.GetString("tado.clientSecret"),
		registry,
	)
	if err != nil {
		return nil, fmt.Errorf("tado: %w", err)
	}

	// Do we have zone rules?
	r, err := maybeLoadRules(filepath.Join(filepath.Dir(cfg.ConfigFileUsed()), "rules.yaml"))
	if err != nil {
		return nil, err
	}
	return taskmanager.New(makeTasks(cfg, api, r, version, registry, logger)...), nil
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

func makeTasks(cfg *viper.Viper, api *tado.APIClient, rules configuration.Configuration, version string, registry prometheus.Registerer, l *slog.Logger) []taskmanager.Task {
	var tasks []taskmanager.Task

	// Poller
	p := poller.New(api, cfg.GetDuration("poller.interval"), l.With("component", "poller"))
	tasks = append(tasks, p)

	// Collector
	coll := &collector.Collector{Poller: p, Logger: l.With("component", "collector")}
	if registry != nil {
		registry.MustRegister(coll)
	}
	tasks = append(tasks, coll)

	// Prometheus Server
	tasks = append(tasks, promserver.New(promserver.WithAddr(cfg.GetString("exporter.addr"))))

	// Health Endpoint
	h := health.New(p, l.With("component", "health"))
	tasks = append(tasks, h)
	r := http.NewServeMux()
	r.Handle("/health", h)
	tasks = append(tasks, httpserver.New(cfg.GetString("health.addr"), r))

	// Controller
	if len(rules.Zones) > 0 {
		tasks = append(tasks, makeControllerTasks(cfg, api, rules, p, version, l)...)
	}
	return tasks
}

func makeControllerTasks(cfg *viper.Viper, api *tado.APIClient, rules configuration.Configuration, p poller.Poller, version string, l *slog.Logger) []taskmanager.Task {
	var tasks []taskmanager.Task

	// Slackbot
	var b *slackbot.SlackBot
	if cfg.GetBool("controller.tadoBot.enabled") {
		if token := cfg.GetString("controller.tadoBot.token"); token != "" {
			b = slackbot.New(
				token,
				slackbot.WithName("tadoBot "+version),
				slackbot.WithLogger(l.With(slog.String("component", "slackbot"))),
			)
			tasks = append(tasks, b)
		}
	}

	c := controller.New(api, rules, b, p, l.With("component", "controller"))
	tasks = append(tasks, c)

	if b != nil {
		tasks = append(tasks, bot.New(api, b, p, c, l.With(slog.String("component", "tadobot"))))
	}
	return tasks
}
