package exporter

import (
	"net/http"
	"tado-exporter/internal/tadoprobe"
	"time"
)

type Configuration struct {
	Username     string
	Password     string
	ClientSecret string
	Interval     time.Duration
	Port         int
	Debug        bool
}

func CreateProbe(cfg *Configuration) *tadoprobe.TadoProbe {
	return &tadoprobe.TadoProbe{
		APIClient: tadoprobe.APIClient{
			HTTPClient: &http.Client{},
			Username:   cfg.Username,
			Password:   cfg.Password,
			Secret:     cfg.ClientSecret,
		},
	}
}

func RunProbe(probe *tadoprobe.TadoProbe, interval time.Duration) {
	for {
		if probe.Run() != nil {
			break
		}
		time.Sleep(interval)
	}
}
