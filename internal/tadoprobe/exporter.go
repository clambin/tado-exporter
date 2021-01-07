package tadoprobe

import (
	"net/http"
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

func CreateProbe(cfg *Configuration) *TadoProbe {
	return &TadoProbe{
		APIClient: APIClient{
			HTTPClient: &http.Client{},
			Username:   cfg.Username,
			Password:   cfg.Password,
			Secret:     cfg.ClientSecret,
		},
	}
}

func RunProbe(probe *TadoProbe, interval time.Duration) {
	for {
		if probe.Run() != nil {
			break
		}
		time.Sleep(interval)
	}
}
