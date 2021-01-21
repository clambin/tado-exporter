package exporter_test

import (
	"github.com/clambin/gotools/metrics"
	"github.com/clambin/tado-exporter/internal/exporter"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testCases = []struct {
	Metric string
	Labels []string
	Value  float64
}{
	{"tado_zone_target_temp_celsius", []string{"Living room", "AUTO"}, 20.0},
	{"tado_zone_target_temp_celsius", []string{"Living room", "MANUAL"}, 0.0},
	{"tado_zone_target_temp_celsius", []string{"Study", "AUTO"}, 0.0},
	{"tado_zone_target_temp_celsius", []string{"Study", "MANUAL"}, 25.0},
	{"tado_zone_power_state", []string{"Living room"}, 1.0},
	{"tado_temperature_celsius", []string{"Living room"}, 19.94},
	{"tado_heating_percentage", []string{"Living room"}, 11.0},
	{"tado_humidity_percentage", []string{"Living room"}, 37.7},
	{"tado_outside_temp_celsius", []string{"Living room"}, 3.4},
	{"tado_solar_intensity_percentage", []string{"Living room"}, 13.3},
	{"tado_open_window_duration", []string{"Living room"}, 50.0},
	{"tado_open_window_remaining", []string{"Living room"}, 250.0},
}

func TestRunProbe(t *testing.T) {
	var err error
	var value float64

	cfg := exporter.Configuration{}
	probe := exporter.CreateProbe(&cfg)
	assert.NotNil(t, probe)

	probe.API = &mockAPI{}

	log.SetLevel(log.DebugLevel)

	err = probe.Run()
	assert.Nil(t, err)

	for _, testCase := range testCases {
		value, err = metrics.LoadValue(testCase.Metric, testCase.Labels...)
		assert.Nil(t, err)
		assert.Equal(t, testCase.Value, value, testCase.Metric)
	}

	value, err = metrics.LoadValue("tado_weather", "CLOUDY_MOSTLY")
	assert.Nil(t, err)
	assert.Equal(t, 1.0, value)

	value, err = metrics.LoadValue("tado_mobile_device_status", "Phone 1")
	assert.Nil(t, err)
	assert.Equal(t, 1.0, value)
	value, err = metrics.LoadValue("tado_mobile_device_status", "Phone 2")
	assert.Nil(t, err)
	assert.Equal(t, 0.0, value)
	value, err = metrics.LoadValue("tado_mobile_device_status", "Phone 3")
	assert.Nil(t, err)
	assert.Equal(t, 0.0, value)
	value, err = metrics.LoadValue("tado_mobile_device_status", "Phone 4")
	// LoadValue doesn't detect non-existing values for the label, so this will succeed
	assert.Nil(t, err)
	assert.Equal(t, 0.0, value)
}

type mockAPI struct {
}

func (client *mockAPI) GetZones() ([]tado.Zone, error) {
	return []tado.Zone{
		{
			ID:   1,
			Name: "Living room",
			Devices: []tado.Device{
				{
					DeviceType:      "RU02",
					Firmware:        "67.2",
					ConnectionState: tado.ConnectionState{Value: true},
					BatteryState:    "NORMAL",
				},
				{
					DeviceType:      "VA02",
					Firmware:        "57.2",
					ConnectionState: tado.ConnectionState{Value: false},
					BatteryState:    "LOW",
				},
			},
		},
		{
			ID:      2,
			Name:    "Study",
			Devices: nil,
		},
		{
			ID:      3,
			Name:    "Bathroom",
			Devices: nil,
		},
	}, nil
}

func (client *mockAPI) GetZoneInfo(zoneID int) (*tado.ZoneInfo, error) {
	info := tado.ZoneInfo{
		Setting: tado.ZoneInfoSetting{
			Power:       "ON",
			Temperature: tado.Temperature{Celsius: 20.0},
		},
		OpenWindow: tado.ZoneInfoOpenWindow{
			DurationInSeconds:      50,
			RemainingTimeInSeconds: 250,
		},
		ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
			HeatingPower: tado.Percentage{Percentage: 11.0},
		},
		SensorDataPoints: tado.ZoneInfoSensorDataPoints{
			Temperature: tado.Temperature{Celsius: 19.94},
			Humidity:    tado.Percentage{Percentage: 37.7},
		},
	}

	if zoneID != 1 {
		info.Overlay = tado.ZoneInfoOverlay{
			Type: "MANUAL",
			Setting: tado.ZoneInfoOverlaySetting{
				Type:        "HEATING",
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: 25.0},
			},
		}
	}

	return &info, nil
}

func (client *mockAPI) GetWeatherInfo() (*tado.WeatherInfo, error) {
	return &tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 3.4},
		SolarIntensity:     tado.Percentage{Percentage: 13.3},
		WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
	}, nil
}

func (client *mockAPI) GetMobileDevices() ([]tado.MobileDevice, error) {
	return []tado.MobileDevice{
		{
			ID:       1,
			Name:     "Phone 1",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: true},
		},
		{
			ID:       2,
			Name:     "Phone 2",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: false},
		},
		{
			ID:       3,
			Name:     "Phone 3",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: true, AtHome: true},
		},
		{
			ID:       4,
			Name:     "Phone 4",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: false},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: true},
		},
	}, nil
}
