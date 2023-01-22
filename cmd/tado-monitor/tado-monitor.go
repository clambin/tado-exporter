package main

import (
	"context"
	"errors"
	"fmt"
	slackbot2 "github.com/clambin/go-common/slackbot"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/collector"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/health"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	//_ "net/http/pprof"
	"os"
	"os/signal"
)

var (
	configFilename string
	cmd            = cobra.Command{
		Use:   "tado-exporter",
		Short: "exports tado metrics to Prometheus",
		Run:   Main,
	}
)

func Main(_ *cobra.Command, _ []string) {
	slog.Info("tado-monitor starting", "version", version.BuildVersion)

	if viper.GetBool("debug") {
		opts := slog.HandlerOptions{Level: slog.LevelDebug}
		slog.SetDefault(slog.New(opts.NewTextHandler(os.Stdout)))
	}

	if viper.GetBool("exporter.enabled") {
		go runPrometheusServer(viper.GetString("exporter.addr"))

	}

	api := tado.New(
		viper.GetString("tado.username"),
		viper.GetString("tado.password"),
		viper.GetString("tado.clientSecret"),
	)

	p := poller.New(api)

	var tadoBot slackbot.SlackBot
	if viper.GetBool("controller.tadoBot.enabled") {
		tadoBot = slackbot2.New("tado "+version.BuildVersion, viper.GetString("controller.tadoBot.token"), nil)
	}

	coll := collector.New(p)
	prometheus.DefaultRegisterer.MustRegister(coll)

	c := controller.New(api, GetZoneRules(), tadoBot, p)
	h := health.New(p)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(5)
	go func() { defer wg.Done(); p.Run(ctx, viper.GetDuration("poller.interval")) }()
	go func() { defer wg.Done(); coll.Run(ctx) }()
	go func() { defer wg.Done(); _ = tadoBot.Run(ctx) }()
	go func() { defer wg.Done(); c.Run(ctx, viper.GetDuration("controller.interval")) }()
	go func() { defer wg.Done(); h.Run(ctx) }()
	go func() {
		r := http.NewServeMux()
		r.Handle("/health", http.HandlerFunc(h.Handle))
		if err := http.ListenAndServe(viper.GetString("controller.addr"), r); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start health server", err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt

	cancel()
	wg.Wait()
	slog.Info("tado-monitor stopped")

	keys := viper.AllKeys()
	sort.Strings(keys)
	fmt.Println(keys)
}

func runPrometheusServer(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(addr, nil); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start prometheus metrics server", err)
	}
}

func GetZoneRules() []rules.ZoneConfig {
	var config []rules.ZoneConfig
	if zones, ok := viper.GetStringMap("controller")["zones"]; ok {
		for _, zone := range zones.([]interface{}) {
			zoneCfg := zone.(map[string]interface{})
			entry := rules.ZoneConfig{Zone: zoneCfg["zone"].(string)}
			config = append(config, entry)

		}
	}
	return config
}

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("failed to start", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	cmd.Version = version.BuildVersion
	cmd.Flags().StringVar(&configFilename, "config", "", "Configuration file")
	cmd.Flags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	} else {
		viper.AddConfigPath("/etc/solaredge/")
		viper.AddConfigPath("$HOME/.solaredge")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.SetDefault("debug", false)
	viper.SetDefault("tado.username", "")
	viper.SetDefault("tado.password", "")
	viper.SetDefault("tado.clientSecret", "")
	viper.SetDefault("exporter.enabled", true)
	viper.SetDefault("exporter.addr", ":9090")
	viper.SetDefault("poller.interval", 15*time.Second)
	viper.SetDefault("controller.addr", ":8080")
	viper.SetDefault("controller.interval", 5*time.Second)
	viper.SetDefault("controller.tadobot.enabled", true)
	viper.SetDefault("controller.tadobot.token", "")

	viper.SetEnvPrefix("TADO")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("failed to read config file", err)
		os.Exit(1)
	}
}
