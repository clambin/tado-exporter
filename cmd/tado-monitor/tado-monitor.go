package main

import (
	"context"
	"errors"
	slackbot2 "github.com/clambin/go-common/slackbot"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/collector"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/health"
	"github.com/clambin/tado-exporter/poller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
	"net/http"
	//_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	configFilename string
	cmd            = cobra.Command{
		Use:     "tado-monitor",
		Short:   "exporter / controller for Tadoº thermostats",
		Run:     Main,
		Version: version,
	}
)

// overridden during build
var version = "change-me"

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}
}

func Main(cmd *cobra.Command, _ []string) {
	var opts slog.HandlerOptions
	if viper.GetBool("debug") {
		opts.Level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &opts)))

	slog.Info("tado-monitor starting", "version", cmd.Version)

	// context to terminate the created go routines
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup

	// Poller
	api := tado.New(
		viper.GetString("tado.username"),
		viper.GetString("tado.password"),
		viper.GetString("tado.clientSecret"),
	)

	p := poller.New(api)
	wg.Add(1)
	go func() { defer wg.Done(); p.Run(ctx, viper.GetDuration("poller.interval")) }()

	// Collector
	coll := collector.Collector{Poller: p}
	prometheus.DefaultRegisterer.MustRegister(&coll)
	wg.Add(1)
	go func() { defer wg.Done(); coll.Run(ctx) }()
	go runPrometheusServer()

	// Health endpoint
	go runHealthEndpoint(ctx, p, &wg)

	// Do we have zone rules?
	r, err := GetZoneRules()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Error("failed to read zone rules", "err", err)
		os.Exit(1)
	}

	if len(r) > 0 {
		go runController(ctx, p, api, r, &wg)
	} else {
		slog.Warn("no rules found. controller will not run")
	}

	<-ctx.Done()

	slog.Info("tado-monitor shutting down")
	wg.Wait()
	slog.Info("tado-monitor stopped")
}

func runPrometheusServer() {
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(viper.GetString("exporter.addr"), nil); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start prometheus metrics server", "err", err)
	}
}

func runHealthEndpoint(ctx context.Context, p poller.Poller, wg *sync.WaitGroup) {
	h := health.New(p)
	wg.Add(1)
	go func() { defer wg.Done(); h.Run(ctx) }()

	handler := http.NewServeMux()
	handler.Handle("/health", http.HandlerFunc(h.Handle))
	if err := http.ListenAndServe(viper.GetString("health.addr"), handler); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start health server", "err", err)
	}
}

func runController(ctx context.Context, p poller.Poller, tadoClient *tado.APIClient, r []rules.ZoneConfig, wg *sync.WaitGroup) {
	// slack bot
	var tadoBot slackbot.SlackBot
	if viper.GetBool("controller.tadoBot.enabled") {
		tadoBot = slackbot2.New(viper.GetString("controller.tadoBot.token"), slackbot2.WithName("tado "+version))
		wg.Add(1)
		go func() { defer wg.Done(); tadoBot.Run(ctx) }()
	}

	// controller
	c := controller.New(tadoClient, r, tadoBot, p)
	wg.Add(1)
	c.Run(ctx)
	wg.Done()
}

func GetZoneRules() ([]rules.ZoneConfig, error) {
	f, err := os.Open(filepath.Join(filepath.Dir(viper.ConfigFileUsed()), "rules.yaml"))
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	var config struct {
		Zones []rules.ZoneConfig `yaml:"zones"`
	}

	if err = yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}
	for _, zone := range config.Zones {
		var kinds []string
		for _, rule := range zone.Rules {
			kinds = append(kinds, rule.Kind.String())
		}
		slog.Info("zone rules found", "zone", zone.Zone, "rules", strings.Join(kinds, ","))
	}
	return config.Zones, nil
}

func init() {
	cobra.OnInitialize(initConfig)
	cmd.Flags().StringVar(&configFilename, "config", "", "Configuration file")
	cmd.Flags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	} else {
		viper.AddConfigPath("/etc/tado-monitor/")
		viper.AddConfigPath("$HOME/.tado-monitor")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.SetDefault("debug", false)
	viper.SetDefault("tado.username", "")
	viper.SetDefault("tado.password", "")
	viper.SetDefault("tado.clientSecret", "")
	viper.SetDefault("exporter.addr", ":9090")
	viper.SetDefault("poller.interval", 30*time.Second)
	viper.SetDefault("health.addr", ":8080")
	viper.SetDefault("controller.tadobot.enabled", true)
	viper.SetDefault("controller.tadobot.token", "")

	viper.SetEnvPrefix("TADO_MONITOR")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("failed to read config file", "err", err)
		os.Exit(1)
	}
}
