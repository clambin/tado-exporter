package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"tado-exporter/internal/exporter"
	"tado-exporter/internal/version"
)

func main() {
	cfg := exporter.Configuration{}

	a := kingpin.New(filepath.Base(os.Args[0]), "media monitor")

	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.Debug)
	a.Flag("port", "API listener port").Default("8080").IntVar(&cfg.Port)
	a.Flag("interval", "Scrape interval").Default("1m").DurationVar(&cfg.Interval)

	cfg.Username = os.Getenv("TADO_USERNAME")
	cfg.Password = os.Getenv("TADO_PASSWORD")
	cfg.ClientSecret = os.Getenv("TADO_CLIENT_SECRET")

	if cfg.Username == "" || cfg.Password == "" {
		log.Error("TADO_USERNAME and/or TASO_PASSWORD environment variables are missing")
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	go func() {
		exporter.RunProbe(exporter.CreateProbe(&cfg), cfg.Interval)
	}()

	listenAddress := fmt.Sprintf(":%d", cfg.Port)
	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(listenAddress, nil)
}
