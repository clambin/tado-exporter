package exporter

import (
	"time"
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
