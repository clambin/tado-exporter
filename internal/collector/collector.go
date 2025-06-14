package collector

import (
	"context"
	"log/slog"
	"sync/atomic"

	"codeberg.org/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
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
			Help:        "Current weather, if the value is one. See label 'tado_weather'",
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
			Help:        "State of the home, if the value is 1. Label home_state specifies the state",
			ConstLabels: nil,
		}, []string{"home_state"}),
	}
}

type Collector struct {
	Poller        *poller.Poller
	Metrics       *Metrics
	Logger        *slog.Logger
	weatherStates atomic.Value
	homeStates    atomic.Value
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
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoMobileDeviceStatus.WithLabelValues(*userInfo.Name).Set(value)
	}
}

func (c *Collector) collectWeather(update poller.Update) {
	weatherStates, ok := c.weatherStates.Load().(set.Set[tado.WeatherState])
	if !ok {
		weatherStates = make(set.Set[tado.WeatherState])
	}
	weatherStates.Add(*update.Weather.WeatherState.Value)
	c.weatherStates.Store(weatherStates)

	for weatherState := range weatherStates {
		var value float64
		if weatherState == *update.Weather.WeatherState.Value {
			value = 1
		}
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoOutsideWeather.WithLabelValues(string(weatherState)).Set(value)
	}

	//goland:noinspection GoMaybeNil
	c.Metrics.tadoOutsideSolarIntensity.WithLabelValues().Set(float64(*update.Weather.SolarIntensity.Percentage))
	//goland:noinspection GoMaybeNil
	c.Metrics.tadoOutsideTemperature.WithLabelValues().Set(float64(*update.Weather.OutsideTemperature.Celsius))
}

func (c *Collector) collectHomeState(home poller.Update) {
	//goland:noinspection GoMaybeNil
	c.Metrics.tadoHomeState.WithLabelValues(string(*home.HomeState.Presence)).Set(1)

	homeStates, ok := c.homeStates.Load().(set.Set[tado.HomePresence])
	if !ok {
		homeStates = make(set.Set[tado.HomePresence])
	}
	homeStates.Add(*home.HomeState.Presence)
	c.homeStates.Store(homeStates)

	for homeState := range homeStates {
		var value float64
		if homeState == *home.HomeState.Presence {
			value = 1
		}
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoHomeState.WithLabelValues(string(homeState)).Set(value)
	}
}

func (c *Collector) collectZones(update poller.Update) {
	for _, zone := range update.Zones {
		c.collectZoneDevices(zone)
		c.collectZoneInfo(zone)
	}
}

func (c *Collector) collectZoneDevices(zone poller.Zone) {
	for _, device := range *zone.Zone.Devices {
		zoneName := *zone.Zone.Name
		deviceType := *device.DeviceType
		id := zoneName + "_" + *device.SerialNo

		var value float64
		if *device.ConnectionState.Value {
			value = 1.0
		}
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoZoneDeviceConnectionStatus.WithLabelValues(zoneName, id, deviceType, *device.CurrentFwVersion).Set(value)

		value = 0.0
		if device.BatteryState != nil && *device.BatteryState == tado.BatteryStateNORMAL {
			value = 1.0
		}
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoZoneDeviceBatteryStatus.WithLabelValues(zoneName, id, deviceType).Set(value)
	}
}

func (c *Collector) collectZoneInfo(zone poller.Zone) {
	zoneName := *zone.Zone.Name
	if zone.ZoneState.SensorDataPoints.InsideTemperature != nil {
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoZoneTemperatureCelsius.WithLabelValues(zoneName).Set(float64(*zone.ZoneState.SensorDataPoints.InsideTemperature.Celsius))
	}
	//goland:noinspection GoMaybeNil
	c.Metrics.tadoZoneTargetTempCelsius.WithLabelValues(zoneName).Set(float64(zone.GetTargetTemperature()))
	if zone.ZoneState.ActivityDataPoints.HeatingPower != nil {
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoZoneHeatingPercentage.WithLabelValues(zoneName).Set(float64(*zone.ZoneState.ActivityDataPoints.HeatingPower.Percentage))
	}
	if zone.ZoneState.SensorDataPoints.Humidity != nil {
		//goland:noinspection GoMaybeNil
		c.Metrics.tadoZoneHumidityPercentage.WithLabelValues(zoneName).Set(float64(*zone.ZoneState.SensorDataPoints.Humidity.Percentage))
	}
	var duration, remaining float64
	if zone.ZoneState.OpenWindow != nil {
		duration = float64(*zone.ZoneState.OpenWindow.DurationInSeconds)
		remaining = float64(*zone.ZoneState.OpenWindow.RemainingTimeInSeconds)
	}
	//goland:noinspection GoMaybeNil
	c.Metrics.tadoZoneOpenWindowDuration.WithLabelValues(zoneName).Set(duration)
	//goland:noinspection GoMaybeNil
	c.Metrics.tadoZoneOpenWindowRemaining.WithLabelValues(zoneName).Set(remaining)

	var value float64
	if *zone.ZoneState.Setting.Power == tado.PowerON {
		value = 1.0
	}
	//goland:noinspection GoMaybeNil
	c.Metrics.tadoZonePowerState.WithLabelValues(zoneName).Set(value)
	value = 0
	if zone.ZoneState.Overlay != nil {
		value = 1
	}
	//goland:noinspection GoMaybeNil
	c.Metrics.tadoZoneTargetManualMode.WithLabelValues(zoneName).Set(value)
}
