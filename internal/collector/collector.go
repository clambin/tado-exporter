package collector

import (
	"context"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
)

var _ prometheus.Collector = &Metrics{}

type Metrics struct {
	tadoZoneDeviceBatteryStatus    *prometheus.GaugeVec
	tadoZoneDeviceConnectionStatus *prometheus.GaugeVec
	tadoMobileDeviceStatus         *prometheus.GaugeVec
	tadoZoneTargetTempCelsius      *prometheus.GaugeVec
	tadoZoneTargetManualMode       *prometheus.GaugeVec
	tadoZonePowerState             *prometheus.GaugeVec
	tadoZoneTemperatureCelsius     *prometheus.GaugeVec
	tadoZoneHeatingPercentage      *prometheus.GaugeVec
	tadoZoneHumidityPercentage     *prometheus.GaugeVec
	tadoOutsideTemperature         *prometheus.GaugeVec
	tadoOutsideSolarIntensity      *prometheus.GaugeVec
	tadoOutsideWeather             *prometheus.GaugeVec
	tadoZoneOpenWindowDuration     *prometheus.GaugeVec
	tadoZoneOpenWindowRemaining    *prometheus.GaugeVec
	tadoHomeState                  *prometheus.GaugeVec
}

func (m Metrics) Describe(ch chan<- *prometheus.Desc) {
	m.tadoMobileDeviceStatus.Describe(ch)
	m.tadoOutsideSolarIntensity.Describe(ch)
	m.tadoOutsideTemperature.Describe(ch)
	m.tadoOutsideWeather.Describe(ch)
	m.tadoZoneDeviceBatteryStatus.Describe(ch)
	m.tadoZoneDeviceConnectionStatus.Describe(ch)
	m.tadoZoneHeatingPercentage.Describe(ch)
	m.tadoZoneHumidityPercentage.Describe(ch)
	m.tadoZoneOpenWindowDuration.Describe(ch)
	m.tadoZoneOpenWindowRemaining.Describe(ch)
	m.tadoZonePowerState.Describe(ch)
	m.tadoZoneTargetManualMode.Describe(ch)
	m.tadoZoneTargetTempCelsius.Describe(ch)
	m.tadoZoneTemperatureCelsius.Describe(ch)
	m.tadoHomeState.Describe(ch)
}

func (m Metrics) Collect(ch chan<- prometheus.Metric) {
	m.tadoMobileDeviceStatus.Collect(ch)
	m.tadoOutsideSolarIntensity.Collect(ch)
	m.tadoOutsideTemperature.Collect(ch)
	m.tadoOutsideWeather.Collect(ch)
	m.tadoZoneDeviceBatteryStatus.Collect(ch)
	m.tadoZoneDeviceConnectionStatus.Collect(ch)
	m.tadoZoneHeatingPercentage.Collect(ch)
	m.tadoZoneHumidityPercentage.Collect(ch)
	m.tadoZoneOpenWindowDuration.Collect(ch)
	m.tadoZoneOpenWindowRemaining.Collect(ch)
	m.tadoZonePowerState.Collect(ch)
	m.tadoZoneTargetManualMode.Collect(ch)
	m.tadoZoneTargetTempCelsius.Collect(ch)
	m.tadoZoneTemperatureCelsius.Collect(ch)
	m.tadoHomeState.Collect(ch)
}

func NewMetrics() *Metrics {
	return &Metrics{
		tadoZoneDeviceBatteryStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "device_battery_status",
			Help:        "Tado device battery status",
			ConstLabels: nil,
		}, []string{"zone_name", "id", "type"}),
		tadoZoneDeviceConnectionStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "device_connection_status",
			Help:        "Tado device connection status",
			ConstLabels: nil,
		}, []string{"zone_name", "id", "type", "firmware"}),
		tadoMobileDeviceStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "mobile",
			Name:        "device_status",
			Help:        `Tado mobile device status. 1 if the device is "home"`,
			ConstLabels: nil,
		}, []string{"name"}),
		tadoZoneTargetTempCelsius: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "target_temp_celsius",
			Help:        "Target temperature of this zone in degrees celsius",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoZoneTargetManualMode: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "target_manual_mode",
			Help:        "1 if this zone is in manual temp target mode",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoZonePowerState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "power_state",
			Help:        "Power status of this zone",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoZoneTemperatureCelsius: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "temperature_celsius",
			Help:        "Current temperature of this zone in degrees celsius",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoZoneHeatingPercentage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "heating_percentage",
			Help:        "Current heating percentage in this zone in percentage (0-100)",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoZoneHumidityPercentage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "humidity_percentage",
			Help:        "Current humidity percentage in this zone in percentage (0-100)",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoOutsideTemperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "",
			Name:        "outside_temp_celsius",
			Help:        "Current outside temperature in degrees celsius",
			ConstLabels: nil,
		}, nil),
		tadoOutsideSolarIntensity: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "",
			Name:        "solar_intensity_percentage",
			Help:        "Current solar intensity in percentage (0-100)",
			ConstLabels: nil,
		}, nil),
		tadoOutsideWeather: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "",
			Name:        "weather",
			Help:        "Current weather. Always one. See label 'tado_weather'",
			ConstLabels: nil,
		}, []string{"tado_weather"}),
		tadoZoneOpenWindowDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "open_window_duration",
			Help:        "Duration of open window event in seconds",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoZoneOpenWindowRemaining: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "zone",
			Name:        "open_window_remaining",
			Help:        "Remaining duration of open window event in seconds",
			ConstLabels: nil,
		}, []string{"zone_name"}),
		tadoHomeState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   "tado",
			Subsystem:   "home",
			Name:        "state",
			Help:        "State of the home. Always 1. Label home_state specifies the state",
			ConstLabels: nil,
		}, []string{"home_state"}),
	}
}

type Collector struct {
	Poller  poller.Poller
	Metrics *Metrics
	Logger  *slog.Logger
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
			c.process(update)
		}
	}
}

func (c *Collector) process(update poller.Update) {
	c.collectUsers(update)
	c.collectWeather(update)
	c.collectHomeState(update)
	c.collectZones(update)
}

func (c *Collector) collectUsers(update poller.Update) {
	for userInfo := range update.MobileDevices.GeoTrackedDevices() {
		var value float64
		if *userInfo.Location.AtHome {
			value = 1.0
		}
		c.Metrics.tadoMobileDeviceStatus.WithLabelValues(*userInfo.Name).Set(value)
	}
}

func (c *Collector) collectWeather(update poller.Update) {
	c.Metrics.tadoOutsideSolarIntensity.WithLabelValues().Set(float64(*update.SolarIntensity.Percentage))
	c.Metrics.tadoOutsideTemperature.WithLabelValues().Set(float64(*update.OutsideTemperature.Celsius))
	c.Metrics.tadoOutsideWeather.WithLabelValues(string(*update.WeatherState.Value)).Set(1)
}

func (c *Collector) collectHomeState(home poller.Update) {
	c.Metrics.tadoHomeState.WithLabelValues(string(*home.HomeState.Presence)).Set(1)
}

func (c *Collector) collectZones(update poller.Update) {
	for _, zone := range update.Zones {
		c.collectZoneDevices(zone)
		c.collectZoneInfo(zone)
	}
}

func (c *Collector) collectZoneDevices(zone poller.Zone) {
	for _, device := range *zone.Devices {
		zoneName := *zone.Name
		deviceType := *device.DeviceType
		id := zoneName + "_" + *device.SerialNo

		var value float64
		if *device.ConnectionState.Value {
			value = 1.0
		}
		c.Metrics.tadoZoneDeviceConnectionStatus.WithLabelValues(zoneName, id, deviceType, *device.CurrentFwVersion).Set(value)

		value = 0.0
		if device.BatteryState != nil && *device.BatteryState == tado.BatteryStateNORMAL {
			value = 1.0
		}
		c.Metrics.tadoZoneDeviceBatteryStatus.WithLabelValues(zoneName, id, deviceType).Set(value)
	}
}

func (c *Collector) collectZoneInfo(zone poller.Zone) {
	zoneName := *zone.Name
	if zone.SensorDataPoints.InsideTemperature != nil {
		c.Metrics.tadoZoneTemperatureCelsius.WithLabelValues(zoneName).Set(float64(*zone.SensorDataPoints.InsideTemperature.Celsius))
	}
	c.Metrics.tadoZoneTargetTempCelsius.WithLabelValues(zoneName).Set(float64(zone.GetTargetTemperature()))
	if zone.ActivityDataPoints.HeatingPower != nil {
		c.Metrics.tadoZoneHeatingPercentage.WithLabelValues(zoneName).Set(float64(*zone.ActivityDataPoints.HeatingPower.Percentage))
	}
	if zone.SensorDataPoints.Humidity != nil {
		c.Metrics.tadoZoneHumidityPercentage.WithLabelValues(zoneName).Set(float64(*zone.SensorDataPoints.Humidity.Percentage))
	}
	var duration, remaining float64
	if zone.OpenWindow != nil {
		duration = float64(*zone.OpenWindow.DurationInSeconds)
		remaining = float64(*zone.OpenWindow.RemainingTimeInSeconds)
	}
	c.Metrics.tadoZoneOpenWindowDuration.WithLabelValues(zoneName).Set(duration)
	c.Metrics.tadoZoneOpenWindowRemaining.WithLabelValues(zoneName).Set(remaining)

	var value float64
	if *zone.Setting.Power == tado.PowerON {
		value = 1.0
	}
	c.Metrics.tadoZonePowerState.WithLabelValues(zoneName).Set(value)
	value = 0
	if zone.ZoneState.Overlay != nil {
		value = 1
	}
	c.Metrics.tadoZoneTargetManualMode.WithLabelValues(zoneName).Set(value)
}
