package oapi

import (
	"github.com/clambin/tado/v2"
)

var (
	LocationHome = tado.MobileDeviceLocation{AtHome: VarP(true)}
	LocationAway = tado.MobileDeviceLocation{AtHome: VarP(false)}

	TerminationManual = tado.ZoneOverlayTermination{Type: VarP[tado.ZoneOverlayTerminationType](tado.ZoneOverlayTerminationTypeMANUAL)}
	TerminationTimer  = tado.ZoneOverlayTermination{
		Type:                   VarP[tado.ZoneOverlayTerminationType](tado.ZoneOverlayTerminationTypeTIMER),
		RemainingTimeInSeconds: VarP(3600),
	}

	SensorDataPoint = tado.SensorDataPoints{
		Humidity:          &tado.PercentageDataPoint{Percentage: VarP[float32](62)},
		InsideTemperature: &tado.TemperatureDataPoint{Celsius: VarP[float32](21)},
	}
)
