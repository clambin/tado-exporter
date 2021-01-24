package controller

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	"time"
)

type Controller struct {
	tado.API
	Rules        *Rules
	AutoAwayInfo map[int]AutoAwayInfo
}

// Configuration options for tado-exporter
type Configuration struct {
	Username     string
	Password     string
	ClientSecret string
	Interval     time.Duration
	// Port         int
	Debug bool
}

func (controller *Controller) Run() error {
	return controller.AutoAwayRun()
}
