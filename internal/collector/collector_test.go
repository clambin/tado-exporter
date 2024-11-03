package collector

import (
	"embed"
	"encoding/json"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"log/slog"
	"strings"
	"testing"
)

//go:embed testdata/*
var testdataFS embed.FS

func MustUpdate() poller.Update {
	f, err := testdataFS.Open("testdata/update.json")
	if err != nil {
		panic(err)
	}
	var update poller.Update
	if err = json.NewDecoder(f).Decode(&update); err != nil {
		panic(err)
	}
	return update
}

func TestCollector(t *testing.T) {
	m := NewMetrics()
	c := Collector{Poller: nil, Metrics: m, Logger: slog.Default()}

	c.process(MustUpdate())

	require.NoError(t, testutil.CollectAndCompare(m, strings.NewReader(`
# HELP tado_home_state State of the home. Always 1. Label home_state specifies the state
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

# HELP tado_weather Current weather. Always one. See label 'tado_weather'
# TYPE tado_weather gauge
tado_weather{tado_weather="DRIZZLE"} 1

# HELP tado_zone_device_battery_status Tado device battery status
# TYPE tado_zone_device_battery_status gauge
tado_zone_device_battery_status{id="Living room_RU0123456789",type="RU02",zone_name="Living room"} 1
tado_zone_device_battery_status{id="Living room_VA0123456789",type="VA02",zone_name="Living room"} 1

# HELP tado_zone_device_connection_status Tado device connection status
# TYPE tado_zone_device_connection_status gauge
tado_zone_device_connection_status{firmware="215.1",id="Living room_VA0123456789",type="VA02",zone_name="Living room"} 1
tado_zone_device_connection_status{firmware="215.2",id="Living room_RU0123456789",type="RU02",zone_name="Living room"} 1

# HELP tado_zone_heating_percentage Current heating percentage in this zone in percentage (0-100)
# TYPE tado_zone_heating_percentage gauge
tado_zone_heating_percentage{zone_name="Living room"} 0

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
`)))
}

func BenchmarkCollector_process(b *testing.B) {
	m := NewMetrics()
	c := Collector{Poller: nil, Metrics: m, Logger: slog.Default()}
	u := MustUpdate()
	b.ResetTimer()
	for range b.N {
		c.process(u)
	}
}
