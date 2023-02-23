package collector

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/poller/mocks"
	"github.com/prometheus/client_golang/prometheus"
	promGo "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCollector(t *testing.T) {
	ch := make(chan *poller.Update, 1)
	p := mocks.NewPoller(t)
	p.On("Register").Return(ch).Once()
	p.On("Unregister", ch).Return().Once()
	c := New(p)

	r := prometheus.NewRegistry()
	r.MustRegister(c)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); c.Run(ctx) }()

	ch <- &Update

	require.Eventually(t, func() bool {
		c.lock.RLock()
		defer c.lock.RUnlock()
		return c.lastUpdate != nil
	}, time.Second, 10*time.Millisecond)

	metrics, err := r.Gather()
	require.NoError(t, err)

	for _, metric := range metrics {
		t.Run(metric.GetName(), func(t *testing.T) {
			name := filepath.Base(t.Name())
			expected, ok := CollectResult[name]
			if !ok {
				t.Logf("unexpected metric '%s'. ignoring ...", metric.GetName())
				return
			}
			for _, m := range metric.GetMetric() {
				expect := expected
				if expected.multiKey != "" {
					for _, l := range m.GetLabel() {
						if l.GetName() == expected.multiKey {
							expect = expected.multiValues[l.GetValue()]
							break
						}
					}
				}
				testMetricResult(t, expect, m, metric.GetType())
			}
		})
	}

	cancel()
	wg.Wait()
}

func testMetricResult(t *testing.T, expected MetricResult, metric *promGo.Metric, metricType promGo.MetricType) {
	t.Helper()
	switch metricType {
	case promGo.MetricType_GAUGE:
		require.NotNil(t, metric.Gauge)
		assert.Equal(t, expected.value, metric.Gauge.GetValue())
	default:
		t.Logf("unsupported metric type: %s", metricType.String())
		t.Fail()
	}

	var matchedLabels int
	for _, l := range metric.GetLabel() {
		if expectedLabel, ok := expected.labels[l.GetName()]; ok {
			assert.Equal(t, expectedLabel, l.GetValue())
			matchedLabels++
		}
	}
	assert.Equal(t, len(expected.labels), matchedLabels)
}

var Update = poller.Update{
	Home: true,
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
					DeviceType:       "RU02",
					CurrentFwVersion: "67.2",
					ConnectionState:  tado.State{Value: true},
					BatteryState:     "NORMAL",
				},
			},
		},
		2: {
			ID:   2,
			Name: "bar",
			Devices: []tado.Device{
				{
					DeviceType:       "VA02",
					CurrentFwVersion: "57.2",
					ConnectionState:  tado.State{Value: false},
					BatteryState:     "LOW",
				},
			},
		},
	},
	ZoneInfo: map[int]tado.ZoneInfo{
		1: {
			Setting: tado.ZonePowerSetting{
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 22.0},
			},
			ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
				HeatingPower: tado.Percentage{Percentage: 85.0},
			},
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{
				InsideTemperature: tado.Temperature{Celsius: 21.0},
				Humidity:          tado.Percentage{Percentage: 65.0},
			},
		},
		2: {
			Setting: tado.ZonePowerSetting{
				Power:       "OFF",
				Temperature: tado.Temperature{Celsius: 25.0},
			},
			ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
				HeatingPower: tado.Percentage{Percentage: 50.0},
			},
			SensorDataPoints: tado.ZoneInfoSensorDataPoints{
				InsideTemperature: tado.Temperature{Celsius: 18.0},
				Humidity:          tado.Percentage{Percentage: 45.0},
			},
			Overlay: tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZonePowerSetting{
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

type MetricResult struct {
	value       float64
	labels      map[string]string
	multiKey    string
	multiValues map[string]MetricResult
}

var CollectResult = map[string]MetricResult{
	"tado_mobile_device_status": {
		multiKey: "name", multiValues: map[string]MetricResult{"foo": {value: 1.0}, "bar": {value: 0.0}},
	},
	"tado_solar_intensity_percentage": {value: 55.0},
	"tado_outside_temp_celsius":       {value: 18.5},
	"tado_weather":                    {value: 1.0, labels: map[string]string{"tado_weather": "SUNNY"}},
	"tado_zone_device_connection_status": {
		multiKey: "zone_name",
		multiValues: map[string]MetricResult{
			"foo": {value: 1.0, labels: map[string]string{"type": "RU02", "firmware": "67.2"}},
			"bar": {value: 0.0, labels: map[string]string{"type": "VA02", "firmware": "57.2"}},
		},
	},
	"tado_zone_device_battery_status": {
		multiKey: "zone_name",
		multiValues: map[string]MetricResult{
			"foo": {value: 1.0, labels: map[string]string{"type": "RU02"}},
			"bar": {value: 0.0, labels: map[string]string{"type": "VA02"}},
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
	"tado_home_state": {value: 1.0, labels: map[string]string{"home_state": "HOME"}},
}
