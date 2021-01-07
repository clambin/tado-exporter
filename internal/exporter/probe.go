package exporter

import (
	log "github.com/sirupsen/logrus"

	"tado-exporter/pkg/tado"
)

// Probe structure representing a tado-exporter probe
type Probe struct {
	tado.APIClient
}

// Run a tado-exporter probe once
func (probe *Probe) Run() error {
	var (
		err             error
		zones           []tado.Zone
		info            *tado.ZoneInfo
		tadoWeatherInfo *tado.WeatherInfo
	)

	if tadoWeatherInfo, err = probe.GetWeatherInfo(); err == nil {
		tadoOutsideTemperature.Set(tadoWeatherInfo.OutsideTemperature.Celsius)
		tadoSolarIntensity.Set(tadoWeatherInfo.SolarIntensity.Percentage)
		tadoWeather.WithLabelValues(tadoWeatherInfo.WeatherState.Value).Set(1)

		log.WithFields(log.Fields{"err": err, "info": tadoWeatherInfo}).Debug("retrieved weather info")
	}

	if err == nil {
		if zones, err = probe.GetZones(); err == nil {
			for _, zone := range zones {
				logger := log.WithFields(log.Fields{"err": err, "zone.ID": zone.ID, "zone.Name": zone.Name})

				if info, err = probe.GetZoneInfo(zone.ID); err == nil {
					tadoZoneTargetTempCelsius.WithLabelValues(zone.Name).Set(info.Setting.Temperature.Celsius)
					powerState := 0.0
					if info.Setting.Power == "ON" {
						powerState = 1.0
					}
					tadoZonePowerState.WithLabelValues(zone.Name).Set(powerState)
					tadoTemperatureCelsius.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Temperature.Celsius)
					tadoHumidityPercentage.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Humidity.Percentage)
					tadoHeatingPercentage.WithLabelValues(zone.Name).Set(info.ActivityDataPoints.HeatingPower.Percentage)

					if info.OpenWindow != "" {
						logger.Infof("openWindow: %s", info.OpenWindow)
					}

					logger.WithField("zoneInfo", info).Debug("retrieved zone info")
				} else {
					break
				}
			}
		}
	}
	if err != nil {
		log.WithField("err", err.Error()).Warning("Failed to get Tado metrics")
	}

	return err
}
