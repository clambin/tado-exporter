package collector

import (
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type deviceConnectionState = struct {
	Timestamp *time.Time `json:"timestamp,omitempty"`
	Value     *bool      `json:"value,omitempty"`
}

func makeUpdate() poller.Update {
	return poller.Update{
		HomeBase: tado.HomeBase{
			Id:   oapi.VarP(tado.HomeId(1)),
			Name: oapi.VarP("My Home"),
		},
		HomeState: tado.HomeState{
			Presence:       oapi.VarP(tado.HOME),
			PresenceLocked: oapi.VarP(false),
		},
		Weather: tado.Weather{
			OutsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP(float32(13))},
			SolarIntensity:     &tado.PercentageDataPoint{Percentage: oapi.VarP(float32(0))},
			WeatherState:       &tado.WeatherStateDataPoint{Value: oapi.VarP(tado.DRIZZLE)},
		},
		Zones: poller.Zones{
			{
				Zone: tado.Zone{
					Id:   oapi.VarP(1),
					Name: oapi.VarP("Living room"),
					Type: nil,
					Devices: &[]tado.DeviceExtra{
						{
							AccessPointWiFi:         nil,
							BatteryState:            oapi.VarP(tado.BatteryStateNORMAL),
							Characteristics:         nil,
							ChildLockEnabled:        nil,
							CommandTableUploadState: nil,
							ConnectionState:         &deviceConnectionState{Value: oapi.VarP(true)},
							CurrentFwVersion:        oapi.VarP("215.2"),
							DeviceType:              oapi.VarP("RU02"),
							Duties:                  nil,
							InPairingMode:           nil,
							IsDriverConfigured:      nil,
							MountingState:           nil,
							MountingStateWithError:  nil,
							Orientation:             nil,
							SerialNo:                oapi.VarP("RU0123456789"),
							ShortSerialNo:           oapi.VarP("RU0123456789"),
						},
					},
				},
				ZoneState: tado.ZoneState{
					ActivityDataPoints: &tado.ActivityDataPoints{
						HeatingPower: &tado.PercentageDataPoint{Percentage: oapi.VarP(float32(25))},
					},
					OpenWindow: &tado.ZoneOpenWindow{
						DurationInSeconds:      oapi.VarP(300),
						RemainingTimeInSeconds: oapi.VarP(150),
					},
					Overlay:     nil,
					OverlayType: nil,
					SensorDataPoints: &tado.SensorDataPoints{
						Humidity:          &tado.PercentageDataPoint{Percentage: oapi.VarP(float32(67.0))},
						InsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP(float32(20))},
					},
					Setting: &tado.ZoneSetting{
						FanLevel:        nil,
						HorizontalSwing: nil,
						IsBoost:         nil,
						Light:           nil,
						Mode:            nil,
						Power:           oapi.VarP(tado.PowerON),
						Temperature:     &tado.Temperature{Celsius: oapi.VarP(float32(18))},
						Type:            nil,
						VerticalSwing:   nil,
					},
				},
			},
		},
		MobileDevices: poller.MobileDevices{
			{
				Id: oapi.VarP(tado.MobileDeviceId(1234567)),
				Location: oapi.VarP(tado.MobileDeviceLocation{
					AtHome: oapi.VarP(true),
					Stale:  oapi.VarP(false),
				}),
				Name: oapi.VarP("owner"),
				Settings: oapi.VarP(tado.MobileDeviceSettings{
					GeoTrackingEnabled: oapi.VarP(true),
				}),
			},
		},
	}
}

func TestCollector(t *testing.T) {
	m := NewMetrics()
	c := Collector{Poller: nil, Metrics: m, Logger: slog.Default()}

	c.process(makeUpdate())

	require.NoError(t, testutil.CollectAndCompare(m, strings.NewReader(`
# HELP tado_home_state State of the home, if the value is 1. Label home_state specifies the state
# TYPE tado_home_state gauge
tado_home_state{home_state="HOME"} 1

# HELP tado_mobile_device_status Tado mobile device status. 1 if the device is "home"
# TYPE tado_mobile_device_status gauge
tado_mobile_device_status{name="owner"} 1

# HELP tado_outside_temp_celsius Current outside temperature in degrees celsius
# TYPE tado_outside_temp_celsius gauge
tado_outside_temp_celsius 13.0

# HELP tado_solar_intensity_percentage Current solar intensity in percentage (0-100)
# TYPE tado_solar_intensity_percentage gauge
tado_solar_intensity_percentage 0

# HELP tado_weather Current weather, if the value is one. See label 'tado_weather'
# TYPE tado_weather gauge
tado_weather{tado_weather="DRIZZLE"} 1

# HELP tado_zone_device_battery_status Tado device battery status
# TYPE tado_zone_device_battery_status gauge
tado_zone_device_battery_status{id="Living room_RU0123456789",type="RU02",zone_name="Living room"} 1

# HELP tado_zone_device_connection_status Tado device connection status
# TYPE tado_zone_device_connection_status gauge
tado_zone_device_connection_status{firmware="215.2",id="Living room_RU0123456789",type="RU02",zone_name="Living room"} 1

# HELP tado_zone_heating_percentage Current heating percentage in this zone in percentage (0-100)
# TYPE tado_zone_heating_percentage gauge
tado_zone_heating_percentage{zone_name="Living room"} 25

# HELP tado_zone_humidity_percentage Current humidity percentage in this zone in percentage (0-100)
# TYPE tado_zone_humidity_percentage gauge
tado_zone_humidity_percentage{zone_name="Living room"} 67.0

# HELP tado_zone_power_state Power status of this zone
# TYPE tado_zone_power_state gauge
tado_zone_power_state{zone_name="Living room"} 1

# HELP tado_zone_target_temp_celsius Target temperature of this zone in degrees celsius
# TYPE tado_zone_target_temp_celsius gauge
tado_zone_target_temp_celsius{zone_name="Living room"} 18

# HELP tado_zone_temperature_celsius Current temperature of this zone in degrees celsius
# TYPE tado_zone_temperature_celsius gauge
tado_zone_temperature_celsius{zone_name="Living room"} 20.0

# HELP tado_zone_open_window_duration Duration of open window event in seconds
# TYPE tado_zone_open_window_duration gauge
tado_zone_open_window_duration{zone_name="Living room"} 300

# HELP tado_zone_open_window_remaining Remaining duration of open window event in seconds
# TYPE tado_zone_open_window_remaining gauge
tado_zone_open_window_remaining{zone_name="Living room"} 150

# HELP tado_zone_target_manual_mode 1 if this zone is in manual temp target mode
# TYPE tado_zone_target_manual_mode gauge
tado_zone_target_manual_mode{zone_name="Living room"} 0
`)))
}

func TestCollector_Weather(t *testing.T) {
	m := NewMetrics()
	c := Collector{Poller: nil, Metrics: m, Logger: slog.Default()}

	update := makeUpdate()
	for _, weather := range []string{"SUN", "CLOUDY", "SNOW"} {
		*update.Weather.WeatherState.Value = tado.WeatherState(weather)
		c.process(update)
	}

	assert.NoError(t, testutil.CollectAndCompare(*m, strings.NewReader(`
# HELP tado_weather Current weather, if the value is one. See label 'tado_weather'
# TYPE tado_weather gauge
tado_weather{tado_weather="CLOUDY"} 0
tado_weather{tado_weather="SNOW"} 1
tado_weather{tado_weather="SUN"} 0
`), "tado_weather"))
}

func TestCollector_HomeState(t *testing.T) {
	m := NewMetrics()
	c := Collector{Poller: nil, Metrics: m, Logger: slog.Default()}

	update := makeUpdate()
	for _, homeState := range []tado.HomePresence{tado.HOME, tado.AWAY} {
		*update.HomeState.Presence = homeState
		c.process(update)
	}

	assert.NoError(t, testutil.CollectAndCompare(*m, strings.NewReader(`
# HELP tado_home_state State of the home, if the value is 1. Label home_state specifies the state
# TYPE tado_home_state gauge
tado_home_state{home_state="AWAY"} 1
tado_home_state{home_state="HOME"} 0
`), "tado_home_state"))
}
