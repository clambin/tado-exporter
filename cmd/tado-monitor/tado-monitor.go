package main

import (
	"context"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/stack"
	"github.com/clambin/tado-exporter/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	// _ "net/http/pprof"
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
	a.Flag("config", "Configuration file").Required().StringVar(&configFile)

	if _, err = a.Parse(os.Args[1:]); err != nil {
		a.Usage(os.Args[1:])
		os.Exit(1)
	}

	log.WithField("version", version.BuildVersion).Info("tado-monitor starting")

	if cfg, err = configuration.LoadConfigurationFile(configFile); err != nil {
		log.Error("Could not load configuration file: " + err.Error())
		os.Exit(2)
	}

	if debug || cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := stack.New(cfg)
	wg := s.Start(ctx)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt

	cancel()
	s.Stop()
	wg.Wait()

	log.Info("tado-monitor exiting")
}