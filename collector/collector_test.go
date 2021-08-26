package collector_test

import (
	"context"
	"github.com/clambin/gotools/metrics"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/collector"
	"github.com/clambin/tado-exporter/poller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCollector_Describe(t *testing.T) {
	c := collector.New()

	ch := make(chan *prometheus.Desc)
	go c.Describe(ch)

	for _, metricName := range []string{
		"tado_mobile_device_status",
		"tado_solar_intensity_percentage",
		"tado_outside_temp_celsius",
		"tado_weather",
		"tado_zone_device_battery_status",
		"tado_zone_device_connection_status",
		"tado_zone_heating_percentage",
		"tado_zone_humidity_percentage",
		"tado_zone_open_window_duration",
		"tado_zone_open_window_remaining",
		"tado_zone_power_state",
		"tado_zone_target_manual_mode",
		"tado_zone_target_temp_celsius",
		"tado_zone_temperature_celsius",
	} {
		metric := <-ch
		assert.Contains(t, metric.String(), "\""+metricName+"\"")
	}
}

func TestCollector_Collect(t *testing.T) {
	c := collector.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)

	c.Update <- &Update

	time.Sleep(100 * time.Millisecond)

	ch := make(chan prometheus.Metric)
	go c.Collect(ch)

	count := countMetricResults(CollectResult)

	for count > 0 {
		m := <-ch
		name := metrics.MetricName(m)

		expected, ok := CollectResult[name]

		if assert.True(t, ok, name) == false {
			continue
		}

		if expected.multiKey != "" {
			key := metrics.MetricLabel(m, expected.multiKey)

			if assert.NotEmpty(t, key) == false {
				continue
			}

			expected, ok = expected.multiValues[key]
			if assert.True(t, ok) == false {
				continue
			}
		}

		assert.Equal(t, expected.value, metrics.MetricValue(m).GetGauge().GetValue(), name)

		for _, labelPair := range expected.labels {
			assert.Equal(t, labelPair.value, metrics.MetricLabel(m, labelPair.name), name)
		}
		count--
	}
}

func BenchmarkCollector_Collect(b *testing.B) {
	c := collector.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)

	c.Update <- &Update
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < 10000; i++ {
		ch := make(chan prometheus.Metric)
		go func(ch chan prometheus.Metric) {
			c.Collect(ch)
			close(ch)
		}(ch)

		for range ch {
		}
	}
}

var Update = poller.Update{
	UserInfo: map[int]tado.MobileDevice{
		1: {
			ID:       1,
			Name:     "foo",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{AtHome: true},
		},
		2: {
			ID:       2,
			Name:     "bar",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{AtHome: false},
		},
	},
	WeatherInfo: tado.WeatherInfo{
		SolarIntensity:     tado.Percentage{Percentage: 55.0},
		OutsideTemperature: tado.Temperature{Celsius: 18.5},
		WeatherState:       tado.Value{Value: "SUNNY"},
	},
	Zones: map[int]tado.Zone{
		1: {
			ID:   1,
			Name: "foo",
			Devices: []tado.Device{
				{
					DeviceType:      "RU02",
					Firmware:        "67.2",
					ConnectionState: tado.ConnectionState{Value: true},
					BatteryState:    "NORMAL",
				},
			},
		},
		2: {
			ID:   2,
			Name: "bar",
			Devices: []tado.Device{
				{
					DeviceType:      "VA02",
					Firmware:        "57.2",
					ConnectionState: tado.ConnectionState{Value: false},
					BatteryState:    "LOW",
				},
			},
		},
	},
	ZoneInfo: map[int]tado.ZoneInfo{
		1: {
			Setting: tado.ZoneInfoSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 22.0},
			},
			ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
				HeatingPower: tado.Percentage{Percentage: 85.0},
			},
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{
				Temperature: tado.Temperature{Celsius: 21.0},
				Humidity:    tado.Percentage{Percentage: 65.0},
			},
		},
		2: {
			Setting: tado.ZoneInfoSetting{
				Power:       "OFF",
				Temperature: tado.Temperature{Celsius: 25.0},
			},
			ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
				HeatingPower: tado.Percentage{Percentage: 50.0},
			},
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{
				Temperature: tado.Temperature{Celsius: 18.0},
				Humidity:    tado.Percentage{Percentage: 45.0},
			},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "???",
					Temperature: tado.Temperature{Celsius: 19.0},
				},
				Termination: tado.ZoneInfoOverlayTermination{
					Type: "MANUAL",
				},
			},
		},
	},
}

var CollectResult = map[string]MetricResult{
	"tado_mobile_device_status": {
		multiKey: "name", multiValues: map[string]MetricResult{"foo": {value: 1.0}, "bar": {value: 0.0}},
	},
	"tado_solar_intensity_percentage": {value: 55.0},
	"tado_outside_temp_celsius":       {value: 18.5},
	"tado_weather":                    {value: 1.0, labels: []LabelPair{{name: "tado_weather", value: "SUNNY"}}},
	"tado_zone_device_connection_status": {
		multiKey: "zone_name",
		multiValues: map[string]MetricResult{
			"foo": {value: 1.0, labels: []LabelPair{{name: "type", value: "RU02"}, {name: "firmware", value: "67.2"}}},
			"bar": {value: 0.0, labels: []LabelPair{{name: "type", value: "VA02"}, {name: "firmware", value: "57.2"}}},
		},
	},
	"tado_zone_device_battery_status": {
		multiKey: "zone_name",
		multiValues: map[string]MetricResult{
			"foo": {value: 1.0, labels: []LabelPair{{name: "type", value: "RU02"}}},
			"bar": {value: 0.0, labels: []LabelPair{{name: "type", value: "VA02"}}},
		},
	},
	"tado_zone_heating_percentage": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 85.0}, "bar": {value: 50.0}},
	},
	"tado_zone_humidity_percentage": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 65.0}, "bar": {value: 45.0}},
	},
	"tado_zone_open_window_duration": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 0.0}, "bar": {value: 0.0}},
	},
	"tado_zone_open_window_remaining": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 0.0}, "bar": {value: 0.0}},
	},
	"tado_zone_power_state": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 1.0}, "bar": {value: 0.0}},
	},
	"tado_zone_target_manual_mode": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 0.0}, "bar": {value: 1.0}},
	},
	"tado_zone_target_temp_celsius": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 22.0}, "bar": {value: 19.0}},
	},
	"tado_zone_temperature_celsius": {
		multiKey: "zone_name", multiValues: map[string]MetricResult{"foo": {value: 21.0}, "bar": {value: 18.0}},
	},
}

type MetricResult struct {
	value       float64
	labels      []LabelPair
	multiKey    string
	multiValues map[string]MetricResult
}

type LabelPair struct {
	name  string
	value string
}

func countMetricResults(results map[string]MetricResult) (count int) {
	for _, result := range results {
		if result.multiKey == "" {
			count++
		} else {
			count += len(result.multiValues)
		}
	}
	return
}
