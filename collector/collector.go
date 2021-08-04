package collector

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

type Collector struct {
	Update                         chan *poller.Update
	lastUpdate                     *poller.Update
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
}

func New() *Collector {
	return &Collector{
		Update: make(chan *poller.Update),
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
			"temperature of this zone in degrees celsius",
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
	}
}

func (collector *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.tadoMobileDeviceStatus
	ch <- collector.tadoOutsideSolarIntensity
	ch <- collector.tadoOutsideTemperature
	ch <- collector.tadoOutsideWeather
	ch <- collector.tadoZoneDeviceBatteryStatus
	ch <- collector.tadoZoneDeviceConnectionStatus
	ch <- collector.tadoZoneHeatingPercentage
	ch <- collector.tadoZoneHumidityPercentage
	ch <- collector.tadoZoneOpenWindowDuration
	ch <- collector.tadoZoneOpenWindowRemaining
	ch <- collector.tadoZonePowerState
	ch <- collector.tadoZoneTargetManualMode
	ch <- collector.tadoZoneTargetTempCelsius
	ch <- collector.tadoZoneTemperatureCelsius
}

func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	log.Debug("prometheus collect called")
	collector.lock.RLock()
	defer collector.lock.RUnlock()

	if collector.lastUpdate != nil {
		collector.collectUsers(ch)
		collector.collectWeather(ch)
		collector.collectZones(ch)
		collector.collectZoneInfos(ch)
	}
}

func (collector *Collector) collectUsers(ch chan<- prometheus.Metric) {
	var value float64
	for _, userInfo := range collector.lastUpdate.UserInfo {
		value = 0.0
		if userInfo.IsHome() == tado.DeviceHome {
			value = 1.0
		}
		ch <- prometheus.MustNewConstMetric(collector.tadoMobileDeviceStatus, prometheus.GaugeValue, value, userInfo.Name)
	}
}

func (collector *Collector) collectWeather(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(collector.tadoOutsideSolarIntensity, prometheus.GaugeValue, collector.lastUpdate.WeatherInfo.SolarIntensity.Percentage)
	ch <- prometheus.MustNewConstMetric(collector.tadoOutsideTemperature, prometheus.GaugeValue, collector.lastUpdate.WeatherInfo.OutsideTemperature.Celsius)
	ch <- prometheus.MustNewConstMetric(collector.tadoOutsideWeather, prometheus.GaugeValue, 1, collector.lastUpdate.WeatherInfo.WeatherState.Value)
}

func (collector *Collector) collectZones(ch chan<- prometheus.Metric) {
	var value float64
	for _, zone := range collector.lastUpdate.Zones {
		for i, device := range zone.Devices {
			id := zone.Name + "_" + strconv.Itoa(i)
			value = 0.0
			if device.ConnectionState.Value == true {
				value = 1.0
			}
			ch <- prometheus.MustNewConstMetric(collector.tadoZoneDeviceConnectionStatus, prometheus.GaugeValue, value, zone.Name, id, device.DeviceType, device.Firmware)

			value = 0.0
			if device.BatteryState == "NORMAL" {
				value = 1.0
			}
			ch <- prometheus.MustNewConstMetric(collector.tadoZoneDeviceBatteryStatus, prometheus.GaugeValue, value, zone.Name, id, device.DeviceType)
		}
	}
}

func (collector *Collector) collectZoneInfos(ch chan<- prometheus.Metric) {
	var value float64
	for zoneID, zoneInfo := range collector.lastUpdate.ZoneInfo {
		zone, ok := collector.lastUpdate.Zones[zoneID]

		if ok == false {
			log.WithField("id", zoneID).Warning("invalid zoneID in collected tado metrics. skipping collection")
			continue
		}

		ch <- prometheus.MustNewConstMetric(collector.tadoZoneHeatingPercentage, prometheus.GaugeValue, zoneInfo.ActivityDataPoints.HeatingPower.Percentage, zone.Name)
		ch <- prometheus.MustNewConstMetric(collector.tadoZoneHumidityPercentage, prometheus.GaugeValue, zoneInfo.SensorDataPoints.Humidity.Percentage, zone.Name)
		ch <- prometheus.MustNewConstMetric(collector.tadoZoneOpenWindowDuration, prometheus.GaugeValue, float64(zoneInfo.OpenWindow.DurationInSeconds), zone.Name)
		ch <- prometheus.MustNewConstMetric(collector.tadoZoneOpenWindowRemaining, prometheus.GaugeValue, float64(zoneInfo.OpenWindow.RemainingTimeInSeconds), zone.Name)

		if zoneInfo.Setting.Power == "ON" {
			value = 1.0
		} else {
			value = 0.0
		}
		ch <- prometheus.MustNewConstMetric(collector.tadoZonePowerState, prometheus.GaugeValue, value, zone.Name)

		if zoneInfo.GetState() == tado.ZoneStateAuto {
			value = 0.0
		} else {
			value = 1.0
		}
		ch <- prometheus.MustNewConstMetric(collector.tadoZoneTargetManualMode, prometheus.GaugeValue, value, zone.Name)

		if zoneInfo.GetState() == tado.ZoneStateAuto {
			value = zoneInfo.Setting.Temperature.Celsius
		} else {
			value = zoneInfo.Overlay.Setting.Temperature.Celsius
		}
		ch <- prometheus.MustNewConstMetric(collector.tadoZoneTargetTempCelsius, prometheus.GaugeValue, value, zone.Name)

		ch <- prometheus.MustNewConstMetric(collector.tadoZoneTemperatureCelsius, prometheus.GaugeValue, zoneInfo.SensorDataPoints.Temperature.Celsius, zone.Name)
	}

}

func (collector *Collector) Run(ctx context.Context) {
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case update := <-collector.Update:
			collector.lock.Lock()
			collector.lastUpdate = update
			collector.lock.Unlock()
		}
	}
}
