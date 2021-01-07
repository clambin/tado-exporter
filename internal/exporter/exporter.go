package exporter

import (
	"net/http"
	"time"

	"tado-exporter/pkg/tado"
)

type Configuration struct {
	Username     string
	Password     string
	ClientSecret string
	Interval     time.Duration
	Port         int
	Debug        bool
}

func CreateProbe(cfg *Configuration) *Probe {
	return &Probe{
		APIClient: tado.APIClient{
			HTTPClient: &http.Client{},
			Username:   cfg.Username,
			Password:   cfg.Password,
			Secret:     cfg.ClientSecret,
		},
	}
}

func RunProbe(probe *Probe, interval time.Duration) {
	for {
		if probe.Run() != nil {
			break
		}
		time.Sleep(interval)
	}
}
