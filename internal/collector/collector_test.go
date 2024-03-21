package collector

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"log/slog"
	"strings"
	"testing"
)

func TestCollector(t *testing.T) {
	m := NewMetrics()
	c := Collector{Poller: nil, Metrics: m, Logger: slog.Default()}

	c.process(Update)

	require.NoError(t, testutil.CollectAndCompare(m, strings.NewReader(`
# HELP tado_home_state State of the home. Always 1. Label home_state specifies the state
# TYPE tado_home_state gauge
tado_home_state{home_state="HOME"} 1
# HELP tado_mobile_device_status Tado mobile device status. 1 if the device is "home"
# TYPE tado_mobile_device_status gauge
tado_mobile_device_status{name="bar"} 0
tado_mobile_device_status{name="foo"} 1
# HELP tado_outside_temp_celsius Current outside temperature in degrees celsius
# TYPE tado_outside_temp_celsius gauge
tado_outside_temp_celsius 18.5
# HELP tado_solar_intensity_percentage Current solar intensity in percentage (0-100)
# TYPE tado_solar_intensity_percentage gauge
tado_solar_intensity_percentage 55
# HELP tado_weather Current weather. Always one. See label 'tado_weather'
# TYPE tado_weather gauge
tado_weather{tado_weather="SUNNY"} 1
# HELP tado_zone_device_battery_status Tado device battery status
# TYPE tado_zone_device_battery_status gauge
tado_zone_device_battery_status{id="bar_0",type="VA02",zone_name="bar"} 0
tado_zone_device_battery_status{id="foo_0",type="RU02",zone_name="foo"} 1
# HELP tado_zone_device_connection_status Tado device connection status
# TYPE tado_zone_device_connection_status gauge
tado_zone_device_connection_status{firmware="57.2",id="bar_0",type="VA02",zone_name="bar"} 0
tado_zone_device_connection_status{firmware="67.2",id="foo_0",type="RU02",zone_name="foo"} 1
# HELP tado_zone_heating_percentage Current heating percentage in this zone in percentage (0-100)
# TYPE tado_zone_heating_percentage gauge
tado_zone_heating_percentage{zone_name="bar"} 50
tado_zone_heating_percentage{zone_name="foo"} 85
# HELP tado_zone_humidity_percentage Current humidity percentage in this zone in percentage (0-100)
# TYPE tado_zone_humidity_percentage gauge
tado_zone_humidity_percentage{zone_name="bar"} 45
tado_zone_humidity_percentage{zone_name="foo"} 65
# HELP tado_zone_open_window_duration Duration of open window event in seconds
# TYPE tado_zone_open_window_duration gauge
tado_zone_open_window_duration{zone_name="bar"} 0
tado_zone_open_window_duration{zone_name="foo"} 0
# HELP tado_zone_open_window_remaining Remaining duration of open window event in seconds
# TYPE tado_zone_open_window_remaining gauge
tado_zone_open_window_remaining{zone_name="bar"} 0
tado_zone_open_window_remaining{zone_name="foo"} 0
# HELP tado_zone_power_state Power status of this zone
# TYPE tado_zone_power_state gauge
tado_zone_power_state{zone_name="bar"} 0
tado_zone_power_state{zone_name="foo"} 1
# HELP tado_zone_target_manual_mode 1 if this zone is in manual temp target mode
# TYPE tado_zone_target_manual_mode gauge
tado_zone_target_manual_mode{zone_name="bar"} 1
tado_zone_target_manual_mode{zone_name="foo"} 0
# HELP tado_zone_target_temp_celsius Target temperature of this zone in degrees celsius
# TYPE tado_zone_target_temp_celsius gauge
tado_zone_target_temp_celsius{zone_name="bar"} 19
tado_zone_target_temp_celsius{zone_name="foo"} 22
# HELP tado_zone_temperature_celsius Current temperature of this zone in degrees celsius
# TYPE tado_zone_temperature_celsius gauge
tado_zone_temperature_celsius{zone_name="bar"} 18
tado_zone_temperature_celsius{zone_name="foo"} 21
`)))
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
