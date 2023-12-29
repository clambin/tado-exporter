package monitor

import (
	"errors"
	"fmt"
	"github.com/clambin/go-common/slackbot"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/go-common/taskmanager/httpserver"
	promserver "github.com/clambin/go-common/taskmanager/prometheus"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/bot"
	"github.com/clambin/tado-exporter/internal/collector"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/health"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func New(cfg *viper.Viper, version string, logger *slog.Logger) (*taskmanager.Manager, error) {
	api, err := tado.New(
		cfg.GetString("tado.username"),
		cfg.GetString("tado.password"),
		cfg.GetString("tado.clientSecret"),
	)
	if err != nil {
		return nil, fmt.Errorf("tado: %w", err)
	}

	// Do we have zone rules?
	r, err := maybeLoadRules(filepath.Join(filepath.Dir(cfg.ConfigFileUsed()), "rules.yaml"), logger)
	if err != nil {
		return nil, err
	}
	return taskmanager.New(makeTasks(cfg, api, r, version, logger)...), nil
}

func maybeLoadRules(path string, logger *slog.Logger) ([]rules.ZoneConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return nil, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	return rules.Load(f, logger)
}

func makeTasks(cfg *viper.Viper, api *tado.APIClient, rules []rules.ZoneConfig, version string, l *slog.Logger) []taskmanager.Task {
	var tasks []taskmanager.Task

	// Poller
	p := poller.New(api, cfg.GetDuration("poller.interval"), l.With("component", "poller"))
	tasks = append(tasks, p)

	// Collector
	coll := &collector.Collector{Poller: p, Logger: l.With("component", "collector")}
	prometheus.MustRegister(coll)
	tasks = append(tasks, coll)

	// Prometheus Server
	tasks = append(tasks, promserver.New(promserver.WithAddr(cfg.GetString("exporter.addr"))))

	// Health Endpoint
	h := health.New(p, l.With("component", "health"))
	tasks = append(tasks, h)
	r := http.NewServeMux()
	r.Handle("/health", h)
	tasks = append(tasks, httpserver.New(cfg.GetString("health.addr"), r))

	// Slackbot
	var b *slackbot.SlackBot
	if token := cfg.GetString("controller.tadoBot.token"); token != "" {
		b = slackbot.New(
			token,
			slackbot.WithName("tadoBot "+version),
			slackbot.WithLogger(l.With(slog.String("component", "slackbot"))),
		)
	}

	var c *controller.Controller
	// Controller
	if len(rules) > 0 {
		c = controller.New(api, rules, b, p, l.With("component", "controller"))
		tasks = append(tasks, c)
	} else {
		l.Warn("no rules found. controller will not run")
	}

	// Slackbot
	if cfg.GetBool("controller.tadoBot.enabled") {
		tasks = append(tasks,
			b,
			bot.New(api, b, p, c.ZoneManagers, l.With(slog.String("component", "tadobot"))),
		)
	}

	return tasks
}
