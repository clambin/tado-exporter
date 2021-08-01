package main

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/collector"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	// _ "net/http/pprof"
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

	ctx, cancel := context.WithCancel(context.Background())

	API := &tado.APIClient{
		HTTPClient:   &http.Client{},
		Username:     username,
		Password:     password,
		ClientSecret: clientSecret,
	}

	p := poller.New(API)
	go p.Run(ctx, cfg.Interval)
	log.WithField("interval", cfg.Interval).Info("poller started")

	if cfg.Exporter.Enabled {
		c := collector.New()
		go c.Run(ctx)

		p.Register <- c.Update
		prometheus.MustRegister(c)
		log.Info("exporter started")
	}

	if cfg.Controller.Enabled {
		var tadoBot *slackbot.SlackBot
		if cfg.Controller.TadoBot.Enabled {
			tadoBot = slackbot.Create("tado "+version.BuildVersion, cfg.Controller.TadoBot.Token.Value, nil)

			go func(ctx context.Context) {
				err2 := tadoBot.Run(ctx)
				if err2 != nil {
					log.WithError(err).Fatal("tadoBot failed to start")
				}
			}(ctx)
		}

		// TODO: can we reuse API?
		API2 := &tado.APIClient{
			HTTPClient:   &http.Client{},
			Username:     username,
			Password:     password,
			ClientSecret: clientSecret,
		}
		var c *controller.Controller
		c, err = controller.New(API2, &cfg.Controller, tadoBot)

		if err != nil {
			log.WithError(err).Fatal("unable to create controller")
		}

		go c.Run(ctx)
		p.Register <- c.Update
		log.Info("controller started")
	}

	go func() {
		listenAddress := fmt.Sprintf(":%d", cfg.Exporter.Port)
		http.Handle("/metrics", promhttp.Handler())
		err = http.ListenAndServe(listenAddress, nil)
		if err != nil {
			log.WithError(err).Fatal("unable to start metrics server")
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt

	cancel()
	time.Sleep(100 * time.Millisecond)
	log.Info("tado-monitor exiting")
}
