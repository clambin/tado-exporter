package exporter

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

// API for the Tado APIClient.
// Used to mock the API during unit testing
type API interface {
	GetZones() ([]tado.Zone, error)
	GetZoneInfo(zoneID int) (*tado.ZoneInfo, error)
	GetWeatherInfo() (*tado.WeatherInfo, error)
	GetMobileDevices() ([]tado.MobileDevice, error)
}

// Probe structure representing a tado-exporter probe
type Probe struct {
	API
	weatherStates map[string]float64
}

// CreateProbe creates a new tado-exporter probe
func CreateProbe(cfg *Configuration) *Probe {
	return &Probe{
		API: &tado.APIClient{
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
	err := probe.runWeather()

	if err == nil {
		err = probe.runMobileDevices()
	}

	if err == nil {
		err = probe.runZones()
	}

	if err != nil {
		log.WithField("err", err).Warning("Failed to get Tado metrics")
	}

	return err
}

func (probe *Probe) runWeather() error {
	var err error
	var weatherInfo *tado.WeatherInfo

	if weatherInfo, err = probe.GetWeatherInfo(); err == nil {
		probe.reportWeather(weatherInfo)
		log.WithField("info", weatherInfo).Debug("retrieved weather info")
	}

	return err
}

func (probe *Probe) reportWeather(weatherInfo *tado.WeatherInfo) {
	for key := range probe.weatherStates {
		probe.weatherStates[key] = 0.0
	}
	probe.weatherStates[weatherInfo.WeatherState.Value] = 1.0
	tadoOutsideTemperature.Set(weatherInfo.OutsideTemperature.Celsius)
	tadoSolarIntensity.Set(weatherInfo.SolarIntensity.Percentage)
	for key, value := range probe.weatherStates {
		tadoWeather.WithLabelValues(key).Set(value)
	}
}

func (probe *Probe) runMobileDevices() error {
	var err error
	var mobileDevices []tado.MobileDevice

	if mobileDevices, err = probe.GetMobileDevices(); err == nil {
		for _, mobileDevice := range mobileDevices {
			probe.reportMobileDevice(&mobileDevice)
			log.WithField("device", mobileDevice.String()).Debug("retrieved mobile device")
		}
	}

	return err
}

func (probe *Probe) reportMobileDevice(mobileDevice *tado.MobileDevice) {
	if mobileDevice.Settings.GeoTrackingEnabled {
		value := 0.0
		if mobileDevice.Location.AtHome && !mobileDevice.Location.Stale {
			value = 1.0
		}
		tadoMobileDeviceStatus.WithLabelValues(mobileDevice.Name).Set(value)
	}
}

func (probe *Probe) runZones() error {
	var (
		err   error
		zones []tado.Zone
		info  *tado.ZoneInfo
	)

	if zones, err = probe.GetZones(); err == nil {
		for _, zone := range zones {
			logger := log.WithFields(log.Fields{"err": err, "zone.ID": zone.ID, "zone.Name": zone.Name})
			if info, err = probe.GetZoneInfo(zone.ID); err == nil {
				probe.reportZone(&zone, info)

				if info.OpenWindow != "" {
					logger.WithField("openWindow", info.OpenWindow).Info("Non-empty openWindow found!")
				}
			} else {
				break
			}
			logger.WithField("zoneInfo", info).Debug("retrieved zone info")
		}
	}

	return err
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
