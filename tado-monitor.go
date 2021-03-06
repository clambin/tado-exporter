package main

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/exporter"
	"github.com/clambin/tado-exporter/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
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

	username := os.Getenv("TADO_USERNAME")
	password := os.Getenv("TADO_PASSWORD")
	clientSecret := os.Getenv("TADO_CLIENT_SECRET")

	if username == "" || password == "" {
		log.Error("TADO_USERNAME/TADO_PASSWORD environment variables not set. Aborting ...")
		os.Exit(1)
	}

	exportTicker := time.NewTicker(cfg.Exporter.Interval)
	defer exportTicker.Stop()

	var export *exporter.Exporter
	if cfg.Exporter.Enabled {
		export = &exporter.Exporter{
			API: &tado.APIClient{
				HTTPClient:   &http.Client{},
				Username:     username,
				Password:     password,
				ClientSecret: clientSecret,
			},
		}
		log.WithFields(log.Fields{
			"interval": cfg.Exporter.Interval,
			"port":     cfg.Exporter.Port,
		}).Info("exporter created")
	}

	controlTicker := time.NewTicker(cfg.Controller.Interval)
	defer controlTicker.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	var control *controller.Controller
	if cfg.Controller.Enabled {
		if control, err = controller.New(username, password, clientSecret, &cfg.Controller); err == nil {
			log.WithField("interval", cfg.Controller.Interval).Info("controller created")

			go control.Run(ctx)
		} else {
			log.WithField("err", err).Fatal("failed to create controller")
		}
	}

	go func() {
		listenAddress := fmt.Sprintf(":%d", cfg.Exporter.Port)
		http.Handle("/metrics", promhttp.Handler())
		_ = http.ListenAndServe(listenAddress, nil)
	}()

	if export != nil {
		if err = export.Run(ctx); err != nil {
			log.WithField("err", err).Warning("exporter failed. Will keep retrying")
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

loop:
	for {
		select {
		case <-exportTicker.C:
			if export != nil {
				if err = export.Run(ctx); err != nil {
					log.WithField("err", err).Warning("exporter failed")
				}
			}
		case <-interrupt:
			break loop
		}
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
	log.Info("tado-monitor exiting")
}
