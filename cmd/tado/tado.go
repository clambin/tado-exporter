package main

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/cmd/cli"
	"github.com/clambin/tado-exporter/internal/cmd/config"
	"github.com/clambin/tado-exporter/internal/cmd/monitor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"log/slog"
	//_ "net/http/pprof"
	"os"
	"os/signal"
)

var (
	// overridden during build
	version   = "change-me"
	configCmd = cobra.Command{
		Use:     "config",
		Short:   "Show Tado configuration",
		RunE:    showConfig,
		Version: version,
	}
	monitorCmd = cobra.Command{
		Use:     "monitor",
		Short:   "Monitor Tado thermostats",
		RunE:    runMonitor,
		Version: version,
	}
)

func main() {
	//go func() { _ = http.ListenAndServe(":6060", nil) }()

	cli.RootCmd.Version = version
	cli.RootCmd.AddCommand(&configCmd, &monitorCmd)

	if err := cli.RootCmd.Execute(); err != nil {
		slog.Error("failed to start", "err", err)
		os.Exit(1)
	}
}

func runMonitor(cmd *cobra.Command, _ []string) error {
	var opts slog.HandlerOptions
	if viper.GetBool("debug") {
		opts.Level = slog.LevelDebug
	}
	l := slog.New(slog.NewJSONHandler(os.Stderr, &opts))

	m, err := monitor.New(viper.GetViper(), version, l)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	prometheus.MustRegister(m)

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	l.Info("tado monitor starting", "version", version)
	defer l.Info("tado monitor stopped")

	return m.Manager.Run(ctx)
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
