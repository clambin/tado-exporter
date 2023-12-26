package main

import (
	"context"
	"github.com/clambin/tado-exporter/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	//_ "net/http/pprof"
	"os"
	"os/signal"
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
	l := slog.New(slog.NewJSONHandler(os.Stderr, &opts))

	l.Info("tado-monitor starting", "version", cmd.Version)

	a, err := app.New(viper.GetViper(), version, l)
	if err != nil {
		l.Error("failed to start", "err", err)
		return
	}

	// context to terminate the created go routines
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err = a.Run(ctx); err != nil {
		l.Error("failed to start tado-monitor", "err", err)
	}

	l.Info("tado-monitor stopped")
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
