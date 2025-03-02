package monitor

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/go-common/charmer"
	"github.com/clambin/go-common/httputils"
	"github.com/clambin/tado-exporter/internal/bot"
	"github.com/clambin/tado-exporter/internal/collector"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/health"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
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
	api, err := instrumentedTadoClient(
		cmd.Context(),
		viper.GetString("tado.username"), viper.GetString("tado.password"),
		requestCounter,
		requestDuration,
	)
	if err != nil {
		return fmt.Errorf("tado: %w", err)
	}
	prometheus.MustRegister(requestCounter, requestDuration)

	var sc *slack.Client
	token := viper.GetString("slack.token")
	appToken := viper.GetString("slack.app-token")
	if token != "" && appToken != "" {
		sc = slack.New(token, slack.OptionAppLevelToken(appToken))
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	l := charmer.GetLogger(cmd)
	l.Info("tado monitor starting", "version", cmd.Root().Version)
	defer l.Info("tado monitor stopped")

	return run(ctx, l, viper.GetViper(), prometheus.DefaultRegisterer, api, sc)

}

func maybeLoadRules(path string) (controller.Configuration, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return controller.Configuration{}, err
	}
	defer func(f *os.File) { _ = f.Close() }(f)

	var cfg controller.Configuration
	err = yaml.NewDecoder(f).Decode(&cfg)
	return cfg, err
}

type TadoClient interface {
	poller.TadoClient
	bot.TadoClient
}

func run(ctx context.Context, l *slog.Logger, v *viper.Viper, registry prometheus.Registerer, api TadoClient, sc *slack.Client) error {
	var g errgroup.Group

	// pprof
	if pprofAddr := v.GetString("pprof"); pprofAddr != "" {
		go func() {
			_ = http.ListenAndServe(pprofAddr, nil)
		}()
	}

	// poller
	p := poller.New(api, v.GetDuration("poller.interval"), l.With("component", "poller"))

	// collector
	tadoMetrics := collector.NewMetrics()
	registry.MustRegister(tadoMetrics)
	coll := collector.Collector{Poller: p, Metrics: tadoMetrics, Logger: l.With("component", "collector")}
	g.Go(func() error { return coll.Run(ctx) })

	// exporter
	g.Go(func() error {
		return httputils.RunServer(ctx, &http.Server{Addr: v.GetString("exporter.addr"), Handler: promhttp.Handler()})
	})

	// health
	h := health.New(p, v.GetDuration("poller.interval"), l.With("component", "health"))
	g.Go(func() error { h.Run(ctx); return nil })
	g.Go(func() error {
		r := http.NewServeMux()
		r.Handle("/health", h)
		return httputils.RunServer(ctx, &http.Server{Addr: v.GetString("health.addr"), Handler: r})
	})

	// Do we have zone rules?
	rules, err := maybeLoadRules(filepath.Join(filepath.Dir(v.ConfigFileUsed()), "rules.yaml"))
	if err != nil {
		return err
	}

	// Controller
	var c *controller.Controller
	if len(rules.Home) > 0 || len(rules.Zones) > 0 {
		var n notifier.Notifier
		if sc != nil {
			n = &notifier.SlackNotifier{SlackSender: sc, Logger: l.With("component", "notifier", "type", "slack")}
		} else {
			n = notifier.SLogNotifier{Logger: l.With("component", "notifier", "type", "slog")}
		}
		if c, err = controller.New(rules, p, api, n, l.With("component", "controller")); err != nil {
			return fmt.Errorf("controller: %w", err)
		}
		g.Go(func() error { return c.Run(ctx) })
	}

	// TadoBot
	var b *bot.Bot
	if sc != nil {
		smc := socketmode.New(sc,
			socketmode.OptionDebug(false),
			socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
		)
		handler := socketmode.NewSocketmodeHandler(smc)

		b = bot.New(api, handler, p, c, l.With(slog.String("component", "tadobot")))
		g.Go(func() error { return b.Run(ctx) })
	}

	// Now that all dependencies have started, start the poller
	g.Go(func() error { return p.Run(ctx) })

	return g.Wait()
}
