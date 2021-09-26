package main

import (
	"context"
	"github.com/clambin/gotools/metrics"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/collector"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/version"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"sync"
	"time"

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

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	startStack(ctx, cfg, &wg)

	server := metrics.NewServer(cfg.Exporter.Port)
	go func() {
		log.Info("metrics server started")
		err = server.Run()
		if err == http.ErrServerClosed {
			log.Info("metrics server stopped")
		} else {
			log.WithError(err).Fatal("unable to start metrics server")
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt

	_ = server.Shutdown(30 * time.Second)
	cancel()
	wg.Wait()

	log.Info("tado-monitor exiting")
}

func startStack(ctx context.Context, cfg *configuration.Configuration, wg *sync.WaitGroup) {
	username := os.Getenv("TADO_USERNAME")
	password := os.Getenv("TADO_PASSWORD")
	clientSecret := os.Getenv("TADO_CLIENT_SECRET")

	if username == "" || password == "" {
		log.Fatal("TADO_USERNAME/TADO_PASSWORD environment variables not set. Aborting ...")
	}

	API := tado.New(username, password, clientSecret)

	tadoPoller := poller.New(API)
	wg.Add(1)
	go func() {
		tadoPoller.Run(ctx, cfg.Interval)
		wg.Done()
	}()

	if cfg.Exporter.Enabled {
		c := collector.New()
		wg.Add(1)
		go func() {
			c.Run(ctx)
			wg.Done()
		}()

		tadoPoller.Register <- c.Update
		prometheus.MustRegister(c)
	}

	if cfg.Controller.Enabled {
		var tadoBot *slackbot.Agent
		if cfg.Controller.TadoBot.Enabled {
			tadoBot = slackbot.Create("tado "+version.BuildVersion, cfg.Controller.TadoBot.Token.Value, nil)

			wg.Add(1)
			go func(ctx context.Context) {
				err := tadoBot.Run(ctx)
				if err != nil {
					log.WithError(err).Fatal("tadoBot failed to start")
				}
				wg.Done()
			}(ctx)
		}

		c, err := controller.New(API, &cfg.Controller, tadoBot, tadoPoller)

		if err != nil {
			log.WithError(err).Fatal("unable to create controller")
		}

		wg.Add(1)
		go func() {
			c.Run(ctx, time.Minute)
			wg.Done()
		}()
		tadoPoller.Register <- c.Updates
	}
}
