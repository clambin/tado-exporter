package monitor

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/go-common/charmer"
	"github.com/clambin/go-common/slackbot"
	"github.com/clambin/tado-exporter/internal/bot"
	"github.com/clambin/tado-exporter/internal/collector"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/health"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var (
	Cmd = cobra.Command{
		Use:   "monitor",
		Short: "Monitor Tado thermostats",
		RunE:  monitor,
	}
)

func monitor(cmd *cobra.Command, _ []string) error {
	l := charmer.GetLogger(cmd)

	tadoHTTPClientMetrics := tadotools.NewTadoCallMetrics("tado", "monitor", nil)
	prometheus.MustRegister(tadoHTTPClientMetrics)

	api, err := tadotools.GetInstrumentedTadoClient(
		viper.GetString("tado.username"),
		viper.GetString("tado.password"),
		viper.GetString("tado.clientSecret"),
		tadoHTTPClientMetrics,
	)
	if err != nil {
		return fmt.Errorf("tado: %w", err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	l.Info("tado monitor starting", "version", cmd.Root().Version)
	defer l.Info("tado monitor stopped")

	return runMonitor(ctx, l, viper.GetViper(), prometheus.DefaultRegisterer, api, cmd.Root().Version)

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

type tadoClient interface {
	poller.TadoGetter
	bot.TadoSetter
	action.TadoSetter
}

func runMonitor(ctx context.Context, l *slog.Logger, v *viper.Viper, registry prometheus.Registerer, api tadoClient, version string) error {
	var g errgroup.Group

	// poller
	p := poller.New(api, v.GetDuration("poller.interval"), l.With("component", "poller"))

	// collector
	tadoMetrics := collector.NewMetrics()
	registry.MustRegister(tadoMetrics)
	coll := collector.Collector{Poller: p, Metrics: tadoMetrics, Logger: l.With("component", "collector")}
	g.Go(func() error { return coll.Run(ctx) })

	// exporter
	go func() {
		if err := http.ListenAndServe(v.GetString("exporter.addr"), promhttp.Handler()); !errors.Is(err, http.ErrServerClosed) {
			l.Error("Prometheus server error", "error", err)
			panic(err)
		}
	}()

	// health
	h := health.New(p, l.With("component", "health"))
	g.Go(func() error { return h.Run(ctx) })
	go func() {
		r := http.NewServeMux()
		r.Handle("/health", h)
		if err := http.ListenAndServe(v.GetString("health.addr"), r); !errors.Is(err, http.ErrServerClosed) {
			l.Error("Health server error", "error", err)
			panic(err)
		}
	}()

	// Do we have zone rules?
	rules, err := maybeLoadRules(filepath.Join(filepath.Dir(v.ConfigFileUsed()), "rules.yaml"))
	if err != nil {
		return err
	}

	// Controller
	if len(rules.Zones) > 0 {
		// Slackbot
		var s *slackbot.SlackBot
		if token := v.GetString("controller.tadoBot.token"); token != "" {
			s = slackbot.New(
				token,
				slackbot.WithName("tadoBot "+version),
				slackbot.WithLogger(l.With(slog.String("component", "slackbot"))),
			)
			g.Go(func() error { return s.Run(ctx) })
		}

		c := controller.New(api, rules, s, p, l.With("component", "controller"))
		g.Go(func() error { return c.Run(ctx) })

		if s != nil && v.GetBool("controller.tadobot.enabled") {
			b := bot.New(api, s, p, c, l.With(slog.String("component", "tadobot")))
			g.Go(func() error { return b.Run(ctx) })
		}
	}

	// now that all dependencies have started, start the poller
	g.Go(func() error { return p.Run(ctx) })

	return g.Wait()
}
