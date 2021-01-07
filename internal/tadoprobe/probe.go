package tadoprobe

import (
	log "github.com/sirupsen/logrus"
)

type TadoProbe struct {
	APIClient
}

func (probe *TadoProbe) Run() error {
	var (
		err   error
		zones []TadoZone
		info  *TadoZoneInfo
	)

	if zones, err = probe.GetZones(); err == nil {
		for _, zone := range zones {
			logger := log.WithFields(log.Fields{"zone.ID": zone.ID, "zone.Name": zone.Name})

			if info, err = probe.GetZoneInfo(zone.ID); err == nil {
				tadoZoneTargetTempCelsius.WithLabelValues(zone.Name).Set(info.Setting.Temperature.Celsius)
				powerState := 0.0
				if info.Setting.Power == "ON" {
					powerState = 100.0
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

	if err != nil {
		log.WithField("err", err.Error()).Warning("Failed to get Tado metrics")
	}

	return err
}
