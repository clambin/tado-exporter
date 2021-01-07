package main

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"tado-exporter/internal/tadoprobe"
	"time"
)

func main() {
	probe := tadoprobe.TadoProbe{
		APIClient: tadoprobe.APIClient{
			HTTPClient: &http.Client{},
			Username:   "someuser@example.com",
			Password:   "somepassword",
		},
	}

	log.SetLevel(log.DebugLevel)

	for {
		_ = probe.Run()

		time.Sleep(5 * time.Minute)
	}
}
