package collector

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"strconv"
	"sync"
)

var (
	tadoZoneDeviceBatteryStatus = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "device_battery_status"),
		"Tado device battery status",
		[]string{"zone_name", "id", "type"},
		nil,
	)

	tadoZoneDeviceConnectionStatus = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "device_connection_status"),
		"Tado device connection status",
		[]string{"zone_name", "id", "type", "firmware"},
		nil,
	)

	tadoMobileDeviceStatus = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "mobile", "device_status"),
		"Tado mobile device status. 1 if the device is \"home\"",
		[]string{"name"},
		nil,
	)

	tadoZoneTargetTempCelsius = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "target_temp_celsius"),
		"Target temperature of this zone in degrees celsius",
		[]string{"zone_name"},
		nil,
	)
	tadoZoneTargetManualMode = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "target_manual_mode"),
		"1 if this zone is in manual temp target mode",
		[]string{"zone_name"},
		nil,
	)
	tadoZonePowerState = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "power_state"),
		"Power status of this zone",
		[]string{"zone_name"},
		nil,
	)
	tadoZoneTemperatureCelsius = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "temperature_celsius"),
		"Current temperature of this zone in degrees celsius",
		[]string{"zone_name"},
		nil,
	)
	tadoZoneHeatingPercentage = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "heating_percentage"),
		"Current heating percentage in this zone in percentage (0-100)",
		[]string{"zone_name"},
		nil,
	)
	tadoZoneHumidityPercentage = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "humidity_percentage"),
		"Current humidity percentage in this zone",
		[]string{"zone_name"},
		nil,
	)
	tadoOutsideTemperature = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "outside", "temp_celsius"),
		"Current outside temperature in degrees celsius",
		nil,
		nil,
	)
	tadoOutsideSolarIntensity = prometheus.NewDesc(
		// TODO: make consistent. tado_outside_solar_intensity_percentage
		prometheus.BuildFQName("tado", "", "solar_intensity_percentage"),
		"Current solar intensity in percentage (0-100)",
		nil,
		nil,
	)
	tadoOutsideWeather = prometheus.NewDesc(
		// TODO: make consistent. tado_outside_weather
		prometheus.BuildFQName("tado", "", "weather"),
		"Current weather. Always one. See label 'tado_weather'",
		[]string{"tado_weather"},
		nil,
	)
	tadoZoneOpenWindowDuration = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "open_window_duration"),
		"Duration of open window event in seconds",
		[]string{"zone_name"},
		nil,
	)
	tadoZoneOpenWindowRemaining = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "zone", "open_window_remaining"),
		"Remaining duration of open window event in seconds",
		[]string{"zone_name"},
		nil,
	)
	tadoHomeState = prometheus.NewDesc(
		prometheus.BuildFQName("tado", "home", "state"),
		"State of the home. Always 1. Label home_state specifies the state",
		[]string{"home_state"},
		nil,
	)
)

type Collector struct {
	Poller     poller.Poller
	Logger     *slog.Logger
	lock       sync.RWMutex
	lastUpdate *poller.Update
}

func (c *Collector) Run(ctx context.Context) error {
	c.Logger.Debug("started")
	defer c.Logger.Debug("stopped")

	ch := c.Poller.Subscribe()
	defer c.Poller.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			c.lock.Lock()
			c.lastUpdate = update
			c.lock.Unlock()
		}
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- tadoMobileDeviceStatus
	ch <- tadoOutsideSolarIntensity
	ch <- tadoOutsideTemperature
	ch <- tadoOutsideWeather
	ch <- tadoZoneDeviceBatteryStatus
	ch <- tadoZoneDeviceConnectionStatus
	ch <- tadoZoneHeatingPercentage
	ch <- tadoZoneHumidityPercentage
	ch <- tadoZoneOpenWindowDuration
	ch <- tadoZoneOpenWindowRemaining
	ch <- tadoZonePowerState
	ch <- tadoZoneTargetManualMode
	ch <- tadoZoneTargetTempCelsius
	ch <- tadoZoneTemperatureCelsius
	ch <- tadoHomeState
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.lastUpdate != nil {
		c.collectUsers(ch)
		c.collectWeather(ch)
		c.collectZones(ch)
		c.collectZoneInfos(ch)
		c.collectHomeState(ch)
	}
}

func (c *Collector) collectUsers(ch chan<- prometheus.Metric) {
	var value float64
	for _, userInfo := range c.lastUpdate.UserInfo {
		value = 0.0
		if userInfo.IsHome() == tado.DeviceHome {
			value = 1.0
		}
		ch <- prometheus.MustNewConstMetric(tadoMobileDeviceStatus, prometheus.GaugeValue, value, userInfo.Name)
	}
}

func (c *Collector) collectWeather(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(tadoOutsideSolarIntensity, prometheus.GaugeValue, c.lastUpdate.WeatherInfo.SolarIntensity.Percentage)
	ch <- prometheus.MustNewConstMetric(tadoOutsideTemperature, prometheus.GaugeValue, c.lastUpdate.WeatherInfo.OutsideTemperature.Celsius)
	ch <- prometheus.MustNewConstMetric(tadoOutsideWeather, prometheus.GaugeValue, 1, c.lastUpdate.WeatherInfo.WeatherState.Value)
}

func (c *Collector) collectZones(ch chan<- prometheus.Metric) {
	var value float64
	for _, zone := range c.lastUpdate.Zones {
		for i, device := range zone.Devices {
			id := zone.Name + "_" + strconv.Itoa(i)
			value = 0.0
			if device.ConnectionState.Value {
				value = 1.0
			}
			ch <- prometheus.MustNewConstMetric(tadoZoneDeviceConnectionStatus, prometheus.GaugeValue, value, zone.Name, id, device.DeviceType, device.CurrentFwVersion)

			value = 0.0
			if device.BatteryState == "NORMAL" {
				value = 1.0
			}
			ch <- prometheus.MustNewConstMetric(tadoZoneDeviceBatteryStatus, prometheus.GaugeValue, value, zone.Name, id, device.DeviceType)
		}
	}
}

func (c *Collector) collectZoneInfos(ch chan<- prometheus.Metric) {
	var value float64
	for zoneID, zoneInfo := range c.lastUpdate.ZoneInfo {
		zone, found := c.lastUpdate.Zones[zoneID]

		if !found {
			c.Logger.Warn("invalid zoneID in collected tado metrics. skipping collection", "id", zoneID)
			continue
		}

		ch <- prometheus.MustNewConstMetric(tadoZoneHeatingPercentage, prometheus.GaugeValue, zoneInfo.ActivityDataPoints.HeatingPower.Percentage, zone.Name)
		ch <- prometheus.MustNewConstMetric(tadoZoneHumidityPercentage, prometheus.GaugeValue, zoneInfo.SensorDataPoints.Humidity.Percentage, zone.Name)
		ch <- prometheus.MustNewConstMetric(tadoZoneOpenWindowDuration, prometheus.GaugeValue, float64(zoneInfo.OpenWindow.DurationInSeconds), zone.Name)
		ch <- prometheus.MustNewConstMetric(tadoZoneOpenWindowRemaining, prometheus.GaugeValue, float64(zoneInfo.OpenWindow.RemainingTimeInSeconds), zone.Name)

		if zoneInfo.Setting.Power == "ON" {
			value = 1.0
		} else {
			value = 0.0
		}
		ch <- prometheus.MustNewConstMetric(tadoZonePowerState, prometheus.GaugeValue, value, zone.Name)

		zoneState := rules.GetZoneState(zoneInfo)
		if zoneState.Overlay == tado.NoOverlay {
			value = 0.0
		} else {
			value = 1.0
		}
		ch <- prometheus.MustNewConstMetric(tadoZoneTargetManualMode, prometheus.GaugeValue, value, zone.Name)

		// TODO: don't think this is necessary: Setting.Temperature always has the active temp setting, even when in overlay.
		if zoneState.Overlay == tado.NoOverlay {
			value = zoneInfo.Setting.Temperature.Celsius
		} else {
			value = zoneInfo.Overlay.Setting.Temperature.Celsius
		}
		ch <- prometheus.MustNewConstMetric(tadoZoneTargetTempCelsius, prometheus.GaugeValue, value, zone.Name)

		ch <- prometheus.MustNewConstMetric(tadoZoneTemperatureCelsius, prometheus.GaugeValue, zoneInfo.SensorDataPoints.InsideTemperature.Celsius, zone.Name)
	}

}

func (c *Collector) collectHomeState(ch chan<- prometheus.Metric) {
	var label string
	if c.lastUpdate.Home {
		label = "HOME"
	} else {
		label = "AWAY"
	}
	ch <- prometheus.MustNewConstMetric(tadoHomeState, prometheus.GaugeValue, 1, label)
}
