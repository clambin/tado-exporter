package exporter

import (
	"context"
	"github.com/clambin/tado"
	log "github.com/sirupsen/logrus"
	"strconv"
)

// Exporter structure representing a tado-exporter probe
type Exporter struct {
	tado.API
	weatherStates map[string]float64
}

// Run a tado-exporter probe once
func (export *Exporter) Run(ctx context.Context) (err error) {
	err = export.runWeather(ctx)

	if err == nil {
		err = export.runMobileDevices(ctx)
	}

	if err == nil {
		err = export.runZones(ctx)
	}

	return err
}

func (export *Exporter) runWeather(ctx context.Context) error {
	var err error
	var weatherInfo tado.WeatherInfo

	if weatherInfo, err = export.GetWeatherInfo(ctx); err == nil {
		export.reportWeather(weatherInfo)
		log.WithField("info", weatherInfo).Debug("retrieved weather info")
	}

	return err
}

func (export *Exporter) reportWeather(weatherInfo tado.WeatherInfo) {
	if export.weatherStates == nil {
		export.weatherStates = make(map[string]float64)
	}

	for key := range export.weatherStates {
		export.weatherStates[key] = 0.0
	}
	export.weatherStates[weatherInfo.WeatherState.Value] = 1.0
	tadoOutsideTemperature.Set(weatherInfo.OutsideTemperature.Celsius)
	tadoSolarIntensity.Set(weatherInfo.SolarIntensity.Percentage)
	for key, value := range export.weatherStates {
		tadoWeather.WithLabelValues(key).Set(value)
	}
}

func (export *Exporter) runMobileDevices(ctx context.Context) error {
	var err error
	var mobileDevices []tado.MobileDevice

	if mobileDevices, err = export.GetMobileDevices(ctx); err == nil {
		for _, mobileDevice := range mobileDevices {
			export.reportMobileDevice(mobileDevice)
			log.WithField("device", mobileDevice.String()).Debug("retrieved mobile device")
		}
	}

	return err
}

func (export *Exporter) reportMobileDevice(mobileDevice tado.MobileDevice) {
	if mobileDevice.Settings.GeoTrackingEnabled {
		value := 0.0
		if mobileDevice.Location.AtHome && !mobileDevice.Location.Stale {
			value = 1.0
		}
		tadoMobileDeviceStatus.WithLabelValues(mobileDevice.Name).Set(value)
	}
}

func (export *Exporter) runZones(ctx context.Context) error {
	var (
		err   error
		zones []tado.Zone
		info  tado.ZoneInfo
	)

	if zones, err = export.GetZones(ctx); err == nil {
		for _, zone := range zones {
			logger := log.WithFields(log.Fields{"err": err, "zone.ID": zone.ID, "zone.Name": zone.Name})
			if info, err = export.GetZoneInfo(ctx, zone.ID); err == nil {
				export.reportZone(zone, info)
				logger.WithField("zoneInfo", info).Debug("retrieved zone info")
			}
		}
	}

	return err
}

func (export *Exporter) reportZone(zone tado.Zone, info tado.ZoneInfo) {
	tadoZoneTargetTempCelsius.WithLabelValues(zone.Name).Set(info.Setting.Temperature.Celsius)
	manualMode := 0.0
	if info.Overlay.Type == "MANUAL" {
		manualMode = 1.0
	}
	tadoZoneTargetManualMode.WithLabelValues(zone.Name).Set(manualMode)
	powerState := 0.0
	if info.Setting.Power == "ON" {
		powerState = 1.0
	}
	tadoZonePowerState.WithLabelValues(zone.Name).Set(powerState)
	tadoTemperatureCelsius.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Temperature.Celsius)
	tadoHumidityPercentage.WithLabelValues(zone.Name).Set(info.SensorDataPoints.Humidity.Percentage)
	tadoHeatingPercentage.WithLabelValues(zone.Name).Set(info.ActivityDataPoints.HeatingPower.Percentage)
	tadoOpenWindowDuration.WithLabelValues(zone.Name).Set(float64(info.OpenWindow.DurationInSeconds))
	tadoOpenWindowRemaining.WithLabelValues(zone.Name).Set(float64(info.OpenWindow.RemainingTimeInSeconds))

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
