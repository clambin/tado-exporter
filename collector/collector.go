package collector

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	tado2 "github.com/clambin/tado-exporter/tado"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
	"strconv"
	"sync"
)

type Collector struct {
	Update                         chan *poller.Update
	lastUpdate                     *poller.Update
	poller                         poller.Poller
	lock                           sync.RWMutex
	tadoMobileDeviceStatus         *prometheus.Desc
	tadoOutsideSolarIntensity      *prometheus.Desc
	tadoOutsideTemperature         *prometheus.Desc
	tadoOutsideWeather             *prometheus.Desc
	tadoZoneDeviceBatteryStatus    *prometheus.Desc
	tadoZoneDeviceConnectionStatus *prometheus.Desc
	tadoZoneHeatingPercentage      *prometheus.Desc
	tadoZoneHumidityPercentage     *prometheus.Desc
	tadoZoneOpenWindowDuration     *prometheus.Desc
	tadoZoneOpenWindowRemaining    *prometheus.Desc
	tadoZonePowerState             *prometheus.Desc
	tadoZoneTargetManualMode       *prometheus.Desc
	tadoZoneTargetTempCelsius      *prometheus.Desc
	tadoZoneTemperatureCelsius     *prometheus.Desc
	tadoHomeState                  *prometheus.Desc
}

func New(p poller.Poller) *Collector {
	return &Collector{
		poller: p,
		tadoZoneDeviceBatteryStatus: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "device_battery_status"),
			"Tado device battery status",
			[]string{"zone_name", "id", "type"},
			nil,
		),
		tadoZoneDeviceConnectionStatus: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "device_connection_status"),
			"Tado device connection status",
			[]string{"zone_name", "id", "type", "firmware"},
			nil,
		),
		tadoMobileDeviceStatus: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "mobile", "device_status"),
			"Tado mobile device status. 1 if the device is \"home\"",
			[]string{"name"},
			nil,
		),

		tadoZoneTargetTempCelsius: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "target_temp_celsius"),
			"Target temperature of this zone in degrees celsius",
			[]string{"zone_name"},
			nil,
		),
		tadoZoneTargetManualMode: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "target_manual_mode"),
			"1 if this zone is in manual temp target mode",
			[]string{"zone_name"},
			nil,
		),
		tadoZonePowerState: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "power_state"),
			"Power status of this zone",
			[]string{"zone_name"},
			nil,
		),
		tadoZoneTemperatureCelsius: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "temperature_celsius"),
			"Current temperature of this zone in degrees celsius",
			[]string{"zone_name"},
			nil,
		),
		tadoZoneHeatingPercentage: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "heating_percentage"),
			"Current heating percentage in this zone in percentage (0-100)",
			[]string{"zone_name"},
			nil,
		),
		tadoZoneHumidityPercentage: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "humidity_percentage"),
			"Current humidity percentage in this zone",
			[]string{"zone_name"},
			nil,
		),
		tadoOutsideTemperature: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "outside", "temp_celsius"),
			"Current outside temperature in degrees celsius",
			nil,
			nil,
		),
		tadoOutsideSolarIntensity: prometheus.NewDesc(
			// TODO: make consistent. tado_outside_solar_intensity_percentage
			prometheus.BuildFQName("tado", "", "solar_intensity_percentage"),
			"Current solar intensity in percentage (0-100)",
			nil,
			nil,
		),
		tadoOutsideWeather: prometheus.NewDesc(
			// TODO: make consistent. tado_outside_weather
			prometheus.BuildFQName("tado", "", "weather"),
			"Current weather. Always one. See label 'tado_weather'",
			[]string{"tado_weather"},
			nil,
		),
		tadoZoneOpenWindowDuration: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "open_window_duration"),
			"Duration of open window event in seconds",
			[]string{"zone_name"},
			nil,
		),
		tadoZoneOpenWindowRemaining: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "zone", "open_window_remaining"),
			"Remaining duration of open window event in seconds",
			[]string{"zone_name"},
			nil,
		),
		tadoHomeState: prometheus.NewDesc(
			prometheus.BuildFQName("tado", "home", "state"),
			"State of the home. Always 1. Label home_state specifies the stace",
			[]string{"home_state"},
			nil,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.tadoMobileDeviceStatus
	ch <- c.tadoOutsideSolarIntensity
	ch <- c.tadoOutsideTemperature
	ch <- c.tadoOutsideWeather
	ch <- c.tadoZoneDeviceBatteryStatus
	ch <- c.tadoZoneDeviceConnectionStatus
	ch <- c.tadoZoneHeatingPercentage
	ch <- c.tadoZoneHumidityPercentage
	ch <- c.tadoZoneOpenWindowDuration
	ch <- c.tadoZoneOpenWindowRemaining
	ch <- c.tadoZonePowerState
	ch <- c.tadoZoneTargetManualMode
	ch <- c.tadoZoneTargetTempCelsius
	ch <- c.tadoZoneTemperatureCelsius
	ch <- c.tadoHomeState
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
		ch <- prometheus.MustNewConstMetric(c.tadoMobileDeviceStatus, prometheus.GaugeValue, value, userInfo.Name)
	}
}

func (c *Collector) collectWeather(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.tadoOutsideSolarIntensity, prometheus.GaugeValue, c.lastUpdate.WeatherInfo.SolarIntensity.Percentage)
	ch <- prometheus.MustNewConstMetric(c.tadoOutsideTemperature, prometheus.GaugeValue, c.lastUpdate.WeatherInfo.OutsideTemperature.Celsius)
	ch <- prometheus.MustNewConstMetric(c.tadoOutsideWeather, prometheus.GaugeValue, 1, c.lastUpdate.WeatherInfo.WeatherState.Value)
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
			ch <- prometheus.MustNewConstMetric(c.tadoZoneDeviceConnectionStatus, prometheus.GaugeValue, value, zone.Name, id, device.DeviceType, device.CurrentFwVersion)

			value = 0.0
			if device.BatteryState == "NORMAL" {
				value = 1.0
			}
			ch <- prometheus.MustNewConstMetric(c.tadoZoneDeviceBatteryStatus, prometheus.GaugeValue, value, zone.Name, id, device.DeviceType)
		}
	}
}

func (c *Collector) collectZoneInfos(ch chan<- prometheus.Metric) {
	var value float64
	for zoneID, zoneInfo := range c.lastUpdate.ZoneInfo {
		zone, found := c.lastUpdate.Zones[zoneID]

		if !found {
			slog.Warn("invalid zoneID in collected tado metrics. skipping collection", "id", zoneID)
			continue
		}

		ch <- prometheus.MustNewConstMetric(c.tadoZoneHeatingPercentage, prometheus.GaugeValue, zoneInfo.ActivityDataPoints.HeatingPower.Percentage, zone.Name)
		ch <- prometheus.MustNewConstMetric(c.tadoZoneHumidityPercentage, prometheus.GaugeValue, zoneInfo.SensorDataPoints.Humidity.Percentage, zone.Name)
		ch <- prometheus.MustNewConstMetric(c.tadoZoneOpenWindowDuration, prometheus.GaugeValue, float64(zoneInfo.OpenWindow.DurationInSeconds), zone.Name)
		ch <- prometheus.MustNewConstMetric(c.tadoZoneOpenWindowRemaining, prometheus.GaugeValue, float64(zoneInfo.OpenWindow.RemainingTimeInSeconds), zone.Name)

		if zoneInfo.Setting.Power == "ON" {
			value = 1.0
		} else {
			value = 0.0
		}
		ch <- prometheus.MustNewConstMetric(c.tadoZonePowerState, prometheus.GaugeValue, value, zone.Name)

		if tado2.GetZoneState(zoneInfo) == tado2.ZoneStateAuto {
			value = 0.0
		} else {
			value = 1.0
		}
		ch <- prometheus.MustNewConstMetric(c.tadoZoneTargetManualMode, prometheus.GaugeValue, value, zone.Name)

		if tado2.GetZoneState(zoneInfo) == tado2.ZoneStateAuto {
			value = zoneInfo.Setting.Temperature.Celsius
		} else {
			value = zoneInfo.Overlay.Setting.Temperature.Celsius
		}
		ch <- prometheus.MustNewConstMetric(c.tadoZoneTargetTempCelsius, prometheus.GaugeValue, value, zone.Name)

		ch <- prometheus.MustNewConstMetric(c.tadoZoneTemperatureCelsius, prometheus.GaugeValue, zoneInfo.SensorDataPoints.InsideTemperature.Celsius, zone.Name)
	}

}

func (c *Collector) collectHomeState(ch chan<- prometheus.Metric) {
	var label string
	if c.lastUpdate.Home {
		label = "HOME"
	} else {
		label = "AWAY"
	}
	ch <- prometheus.MustNewConstMetric(c.tadoHomeState, prometheus.GaugeValue, 1, label)
}

func (c *Collector) Run(ctx context.Context) {
	slog.Info("exporter started")

	ch := c.poller.Register()
	defer c.poller.Unregister(ch)

	for {
		select {
		case <-ctx.Done():
			slog.Info("exporter stopped")
			return
		case update := <-ch:
			c.lock.Lock()
			c.lastUpdate = update
			c.lock.Unlock()
		}
	}
}
