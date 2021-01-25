package main

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/exporter"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	cfg := exporter.Configuration{
		Username:     os.Getenv("TADO_USERNAME"),
		Password:     os.Getenv("TADO_PASSWORD"),
		ClientSecret: os.Getenv("TADO_CLIENT_SECRET"),
	}

	a := kingpin.New(filepath.Base(os.Args[0]), "tado-exporter")
	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.Debug)
	a.Flag("port", "API listener port").Default("8080").IntVar(&cfg.Port)
	a.Flag("interval", "Scrape interval").Default("1m").DurationVar(&cfg.Interval)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	if cfg.Username == "" || cfg.Password == "" {
		log.Error("TADO_USERNAME and/or TADO_PASSWORD environment variables are missing")
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}
	log.Infof("tado-exporter %s", version.BuildVersion)

	go func() {
		probe := exporter.CreateProbe(&cfg)

		for {
			if err = probe.Run(); err != nil {
				log.WithField("err", err).Warning("Failed to get Tado metrics")
			}

			time.Sleep(cfg.Interval)
		}
	}()

	listenAddress := fmt.Sprintf(":%d", cfg.Port)
	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(listenAddress, nil)
}