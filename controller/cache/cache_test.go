package cache_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/cache"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCache_GetName(t *testing.T) {
	testCache := cache.New()
	testCache.Update(&poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
	})

	name, ok := testCache.GetZoneName(1)
	assert.True(t, ok)
	assert.Equal(t, "foo", name)

	name, ok = testCache.GetZoneName(3)
	assert.False(t, ok)

	name, ok = testCache.GetUserName(2)
	assert.True(t, ok)
	assert.Equal(t, "bar", name)

	name, ok = testCache.GetUserName(3)
	assert.False(t, ok)

	var id int
	id, name, ok = testCache.LookupZone(1, "")
	assert.True(t, ok)
	assert.Equal(t, 1, id)
	assert.Equal(t, "foo", name)

	id, name, ok = testCache.LookupZone(0, "bar")
	assert.True(t, ok)
	assert.Equal(t, 2, id)
	assert.Equal(t, "bar", name)

	id, name, ok = testCache.LookupZone(0, "snafu")
	assert.False(t, ok)

	id, name, ok = testCache.LookupUser(1, "")
	assert.True(t, ok)
	assert.Equal(t, 1, id)
	assert.Equal(t, "foo", name)

	id, name, ok = testCache.LookupUser(0, "bar")
	assert.True(t, ok)
	assert.Equal(t, 2, id)
	assert.Equal(t, "bar", name)

	id, name, ok = testCache.LookupUser(0, "snafu")
	assert.False(t, ok)
}

func TestCache_GetZoneInfo(t *testing.T) {
	testCache := cache.New()
	testCache.Update(&poller.Update{
		Zones: map[int]tado.Zone{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {
				SensorDataPoints: tado.ZoneInfoSensorDataPoints{
					Temperature: tado.Temperature{Celsius: 18.5},
				},
				Setting: tado.ZoneInfoSetting{
					Power:       "ON",
					Temperature: tado.Temperature{Celsius: 22.0},
				},
			},
			2: {
				SensorDataPoints: tado.ZoneInfoSensorDataPoints{
					Temperature: tado.Temperature{Celsius: 22.5},
				},
				Setting: tado.ZoneInfoSetting{
					Power:       "ON",
					Temperature: tado.Temperature{Celsius: 22.0},
				},
				Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 23.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "TIMER", DurationInSeconds: 500},
				},
			},
		},
	})

	temperature, targetTemperature, zoneState, duration, found := testCache.GetZoneInfo(1)
	assert.True(t, found)
	assert.Equal(t, 18.5, temperature)
	assert.Equal(t, 22.0, targetTemperature)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), zoneState)

	temperature, targetTemperature, zoneState, duration, found = testCache.GetZoneInfo(2)
	assert.True(t, found)
	assert.Equal(t, 22.5, temperature)
	assert.Equal(t, 23.0, targetTemperature)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateTemporaryManual), zoneState)
	assert.Equal(t, 500*time.Second, duration)

	_, _, _, _, found = testCache.GetZoneInfo(3)
	assert.False(t, found)

}

func BenchmarkCache_Update(b *testing.B) {
	testCache := cache.New()
	update := &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
	}

	for i := 0; i < 1000000; i++ {
		testCache.Update(update)
	}
}
