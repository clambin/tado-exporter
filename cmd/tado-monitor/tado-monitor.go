package main

import (
	"context"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/stack"
	"github.com/clambin/tado-exporter/version"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
	"gopkg.in/alecthomas/kingpin.v2"
	//_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
)

func main() {
	var (
		err        error
		debug      bool
		configFile string
		cfg        *configuration.Configuration
	)

	a := kingpin.New(filepath.Base(os.Args[0]), "tado-monitor")
	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("debug", "Log debug messages").BoolVar(&debug)
	a.Flag("config", "Configuration file").Required().ExistingFileVar(&configFile)

	if _, err = a.Parse(os.Args[1:]); err != nil {
		a.Usage(os.Args[1:])
		os.Exit(1)
	}

	f, _ := os.Open(configFile)
	defer func() {
		_ = f.Close()
	}()

	if cfg, err = configuration.LoadConfiguration(f); err != nil {
		slog.Error("Could not load configuration file: ", err)
		os.Exit(2)
	}

	var opts slog.HandlerOptions
	if debug || cfg.Debug {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	}
	slog.SetDefault(slog.New(opts.NewTextHandler(os.Stdout)))

	slog.Info("tado-monitor starting", "version", version.BuildVersion)

	s, err := stack.New(cfg)
	if err != nil {
		slog.Error("failed to initialize", err)
		os.Exit(1)
	}
	prometheus.DefaultRegisterer.MustRegister(s.HTTPServer)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// go http.ListenAndServe(":9091", http.DefaultServeMux)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt

	cancel()
	s.Stop()

	slog.Info("tado-monitor exiting")
}
