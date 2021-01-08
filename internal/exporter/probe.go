package exporter

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"

	"tado-exporter/pkg/tado"
)

// Probe structure representing a tado-exporter probe
type Probe struct {
	tado.APIClient
	weatherStates map[string]float64
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
		weatherStates: make(map[string]float64),
	}
}

// Run a tado-exporter probe once
func (probe *Probe) Run() error {
	var (
		err         error
		zones       []tado.Zone
		info        *tado.ZoneInfo
		weatherInfo *tado.WeatherInfo
	)

	if weatherInfo, err = probe.GetWeatherInfo(); err == nil {
		probe.reportWeather(weatherInfo)
		log.WithFields(log.Fields{"err": err, "info": weatherInfo}).Debug("retrieved weather info")
	}

	if err == nil {
		if zones, err = probe.GetZones(); err == nil {
			for _, zone := range zones {
				logger := log.WithFields(log.Fields{"err": err, "zone.ID": zone.ID, "zone.Name": zone.Name})
				if info, err = probe.GetZoneInfo(zone.ID); err == nil {
					probe.reportZone(&zone, info)

					logger.WithField("zoneInfo", info).Debug("retrieved zone info")
					if info.OpenWindow != "" {
						logger.Infof("openWindow: %s", info.OpenWindow)
					}
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

func (probe *Probe) reportWeather(weatherInfo *tado.WeatherInfo) {
	for key := range probe.weatherStates {
		probe.weatherStates[key] = 0.0
	}
	probe.weatherStates[weatherInfo.WeatherState.Value] = 1.0
	log.WithField("states", probe.weatherStates).Debug("weatherStates")
	tadoOutsideTemperature.Set(weatherInfo.OutsideTemperature.Celsius)
	tadoSolarIntensity.Set(weatherInfo.SolarIntensity.Percentage)
	for key, value := range probe.weatherStates {
		tadoWeather.WithLabelValues(key).Set(value)
	}
}

func (probe *Probe) reportZone(zone *tado.Zone, info *tado.ZoneInfo) {
	tadoZoneTargetTempCelsius.WithLabelValues(zone.Name).Set(info.Setting.Temperature.Celsius)
	powerState := 0.0
	if info.Setting.Power == "ON" {
		powerState = 1.0
	}
	tadoZonePowerState.WithLabelValues(zone.Name).Set(powerState)
	tadoTemperatureCelsius.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Temperature.Celsius)
	tadoHumidityPercentage.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Humidity.Percentage)
	tadoHeatingPercentage.WithLabelValues(zone.Name).Set(info.ActivityDataPoints.HeatingPower.Percentage)

	for i, device := range zone.Devices {
		id := zone.Name + "_" + strconv.Itoa(i)
		val := 1.0
		if device.ConnectionState.Value == false {
			val = 0.0
		}
		tadoDeviceConnectionStatus.WithLabelValues(zone.Name, id, device.DeviceType, device.Firmware).Set(val)
		val = 1.0
		if device.BatteryState != "NORMAL" {
			val = 0.0
		}
		tadoDeviceBatteryStatus.WithLabelValues(zone.Name, id, device.DeviceType).Set(val)
	}
}
