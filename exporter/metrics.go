package exporter

import (
	"github.com/clambin/gotools/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	tadoZoneTargetTempCelsius = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_target_temp_celsius",
		Help: "Target temperature of this zone in degrees celsius",
	}, []string{"zone_name"})

	tadoZoneTargetManualMode = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_target_manual_mode",
		Help: "1 if this zone is in manual temp target mode",
	}, []string{"zone_name"})

	tadoZonePowerState = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_power_state",
		Help: "Power status of this zone",
	}, []string{"zone_name"})

	tadoTemperatureCelsius = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_temperature_celsius",
		Help: "Current temperature of this zone in degrees celsius",
	}, []string{"zone_name"})

	tadoHeatingPercentage = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_heating_percentage",
		Help: "Current heating percentage in this zone in percentage (0-100)",
	}, []string{"zone_name"})

	tadoHumidityPercentage = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_humidity_percentage",
		Help: "Current humidity percentage in this zone",
	}, []string{"zone_name"})

	tadoOutsideTemperature = metrics.NewGauge(prometheus.GaugeOpts{
		Name: "tado_outside_temp_celsius",
		Help: "Current outside temperature in degrees celsius",
	})

	tadoSolarIntensity = metrics.NewGauge(prometheus.GaugeOpts{
		Name: "tado_solar_intensity_percentage",
		Help: "Current solar intensity in percentage (0-100)",
	})

	tadoWeather = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_weather",
		Help: "Current weather. Always one. See label 'tado_weather'",
	}, []string{"tado_weather"})

	tadoDeviceConnectionStatus = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_device_connection_status",
		Help: "Tado device connection status",
	}, []string{"zone_name", "id", "type", "firmware"})

	tadoDeviceBatteryStatus = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_device_battery_status",
		Help: "Tado device battery status",
	}, []string{"zone_name", "id", "type"})

	tadoMobileDeviceStatus = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_mobile_device_status",
		Help: "Tado mobile device status. 1 if the device is \"home\"",
	}, []string{"name"})

	tadoOpenWindowDuration = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_open_window_duration",
		Help: "Duration of open window event in seconds",
	}, []string{"zone_name"})

	tadoOpenWindowRemaining = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tado_zone_open_window_remaining",
		Help: "Remaining duration of open window event in seconds",
	}, []string{"zone_name"})
)
