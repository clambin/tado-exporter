package main

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/app"
	"github.com/clambin/tado-exporter/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"log/slog"
	//_ "net/http/pprof"
	"os"
	"os/signal"
	"time"
)

var (
	configFilename string
	rootCmd        = cobra.Command{
		Use:     "tado",
		Short:   "Controller for Tadoº thermostats",
		Version: version,
	}
	configCmd = cobra.Command{
		Use:   "config",
		Short: "Show Tado configuration",
		RunE:  showConfig,
	}
	monitorCmd = cobra.Command{
		Use:   "monitor",
		Short: "Monitor Tado thermostats",
		Run:   monitor,
	}
)

// overridden during build
var version = "change-me"

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}
}

func monitor(cmd *cobra.Command, _ []string) {
	var opts slog.HandlerOptions
	if viper.GetBool("debug") {
		opts.Level = slog.LevelDebug
	}
	l := slog.New(slog.NewJSONHandler(os.Stderr, &opts))

	l.Info("tado monitor starting", "version", cmd.Version)

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

func showConfig(cmd *cobra.Command, _ []string) error {
	api, err := tado.New(
		viper.GetString("tado.username"),
		viper.GetString("tado.password"),
		viper.GetString("tado.clientSecret"),
	)
	if err != nil {
		return fmt.Errorf("tado: %w", err)
	}

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)

	return config.ShowConfig(cmd.Context(), api, enc)
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&configFilename, "config", "", "Configuration file")
	rootCmd.PersistentFlags().Bool("debug", false, "Log debug messages")
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.AddCommand(&configCmd, &monitorCmd)
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
