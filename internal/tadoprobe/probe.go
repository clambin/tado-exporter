package tadoprobe

import log "github.com/sirupsen/logrus"

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
			log.Debugf("Querying zone %d (%s)", zone.ID, zone.Name)

			if info, err = probe.GetZoneInfo(zone.ID); err == nil {
				tadoZoneTargetTempCelsius.WithLabelValues(zone.Name).Set(info.Setting.Temperature.Celsius)
				log.Debugf("tadoZoneTargetTempCelsius(%s): %f", zone.Name, info.Setting.Temperature.Celsius)
				powerState := 0.0
				if info.Setting.Power == "ON" {
					powerState = 100.0
				}
				tadoZonePowerState.WithLabelValues(zone.Name).Set(powerState)
				log.Debugf("tadoZonePowerState(%s): %f", zone.Name, powerState)
				tadoTemperatureCelsius.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Temperature.Celsius)
				log.Debugf("tadoTemperatureCelsius(%s): %f", zone.Name, info.SensorDataPoints.Temperature.Celsius)
				tadoHumidityPercentage.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Humidity.Percentage)
				log.Debugf("tadoHumidityPercentage(%s): %f", zone.Name, info.SensorDataPoints.Humidity.Percentage)
				tadoHeatingPercentage.WithLabelValues(zone.Name).Set(info.ActivityDataPoints.HeatingPower.Percentage)
				log.Debugf("tadoHeatingPercentage(%s): %f", zone.Name, info.ActivityDataPoints.HeatingPower.Percentage)

				log.Debugf("openWindow(%s): %s", zone.Name, info.OpenWindow)
			} else {
				break
			}
		}
	}

	if err != nil {
		log.Warningf("Failed to get state: %s", err.Error())
	}

	return err
}
