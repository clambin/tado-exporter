package exporter

import (
	"net/http"
	"time"

	"tado-exporter/pkg/tado"
)

// Configuration options for tado-exporter
type Configuration struct {
	Username     string
	Password     string
	ClientSecret string
	Interval     time.Duration
	Port         int
	Debug        bool
}

// CreateProbe creates a new tado-exporter probe
func CreateProbe(cfg *Configuration) *Probe {
	return &Probe{
		APIClient: tado.APIClient{
			HTTPClient:   &http.Client{},
			Username:     cfg.Username,
			Password:     cfg.Password,
			ClientSecret: cfg.ClientSecret,
		},
	}
}
