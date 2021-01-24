package main

import (
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var (
		err           error
		rulesFilename string
		rules         *controller.Rules
	)

	cfg := controller.Configuration{
		Username:     os.Getenv("TADO_USERNAME"),
		Password:     os.Getenv("TADO_PASSWORD"),
		ClientSecret: os.Getenv("TADO_CLIENT_SECRET"),
	}

	a := kingpin.New(filepath.Base(os.Args[0]), "tado-exporter")
	a.Version(version.BuildVersion)
	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
	a.Flag("debug", "Log debug messages").BoolVar(&cfg.Debug)
	// a.Flag("port", "API listener port").Default("8080").IntVar(&cfg.Port)
	a.Flag("interval", "Scrape interval").Default("1m").DurationVar(&cfg.Interval)
	a.Flag("rules", "Rules config file").Short('r').Required().StringVar(&rulesFilename)

	_, err = a.Parse(os.Args[1:])
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
	log.Infof("tado-controller %s", version.BuildVersion)

	if rules, err = controller.ParseRulesFile(rulesFilename); err != nil {
		log.WithField("err", err).Error("invalid rules file")
		os.Exit(1)
	}

	c := controller.Controller{
		API: &tado.APIClient{
			HTTPClient:   &http.Client{},
			Username:     cfg.Username,
			Password:     cfg.Password,
			ClientSecret: cfg.ClientSecret,
		},
		Rules: rules,
	}

	for {
		if err = c.Run(); err != nil {
			log.WithField("err", err).Warning("controller failed")
			break
		}

		time.Sleep(cfg.Interval)
	}

	// listenAddress := fmt.Sprintf(":%d", cfg.Port)
	// http.Handle("/metrics", promhttp.Handler())
	// _ = http.ListenAndServe(listenAddress, nil)
}
