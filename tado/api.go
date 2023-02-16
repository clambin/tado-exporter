package tado

import (
	"context"
	"github.com/clambin/tado"
	"time"
)

// API interface for github.com/clambin/tado
//
//go:generate mockery --name API
type API interface {
	GetWeatherInfo(context.Context) (tado.WeatherInfo, error)
	GetMobileDevices(context.Context) ([]tado.MobileDevice, error)
	GetZones(context.Context) (tado.Zones, error)
	GetZoneInfo(context.Context, int) (tado.ZoneInfo, error)
	DeleteZoneOverlay(context.Context, int) error
	SetZoneOverlay(context.Context, int, float64) error
	SetZoneTemporaryOverlay(context.Context, int, float64, time.Duration) error
}
