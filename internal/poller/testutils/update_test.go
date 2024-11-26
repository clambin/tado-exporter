package testutils

import (
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWithHome(t *testing.T) {
	u := Update(WithHome(1, "my home", tado.HOME))
	assert.Equal(t, tado.HomeId(1), *u.HomeBase.Id)
	assert.Equal(t, "my home", *u.HomeBase.Name)
	assert.Equal(t, tado.HOME, *u.HomeState.Presence)
}

func TestWithZone(t *testing.T) {
	u := Update(WithZone(10, "my room", tado.PowerON, 18, 19))
	require.Len(t, u.Zones, 1)
	assert.Equal(t, 10, *u.Zones[0].Id)
	assert.Equal(t, "my room", *u.Zones[0].Name)
	assert.Equal(t, tado.PowerON, *u.Zones[0].ZoneState.Setting.Power)
	assert.Equal(t, float32(18), *u.Zones[0].ZoneState.Setting.Temperature.Celsius)
	assert.Equal(t, float32(19), *u.Zones[0].ZoneState.SensorDataPoints.InsideTemperature.Celsius)
}

func TestWithZoneOverlay(t *testing.T) {
	u := Update(WithZone(10, "my room", tado.PowerON, 18, 19, WithZoneOverlay(tado.ZoneOverlayTerminationTypeTIMER, 300)))
	require.Len(t, u.Zones, 1)
	require.NotNil(t, u.Zones[0].ZoneState.Overlay)
	assert.Equal(t, tado.ZoneOverlayTerminationTypeTIMER, *u.Zones[0].ZoneState.Overlay.Termination.Type)
	assert.Equal(t, 300, *u.Zones[0].ZoneState.Overlay.Termination.RemainingTimeInSeconds)
}

func TestWithMobileDevice(t *testing.T) {
	type want struct {
		id                tado.MobileDeviceId
		name              string
		isGeoTrackEnabled bool
		hasLocation       bool
		isHome            bool
		isStale           bool
	}
	tests := []struct {
		name    string
		options []MobileDeviceOption
		want    want
	}{
		{
			name: "no options",
			want: want{
				id:   tado.MobileDeviceId(100),
				name: "phone",
			},
		},
		{
			name:    "geotracked",
			options: []MobileDeviceOption{WithGeoTracking()},
			want: want{
				id:                tado.MobileDeviceId(100),
				name:              "phone",
				isGeoTrackEnabled: true,
			},
		},
		{
			name:    "location",
			options: []MobileDeviceOption{WithLocation(true, true)},
			want: want{
				id:                tado.MobileDeviceId(100),
				name:              "phone",
				isGeoTrackEnabled: true,
				hasLocation:       true,
				isHome:            true,
				isStale:           true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Update(WithMobileDevice(100, "phone", tt.options...))
			require.Len(t, m.MobileDevices, 1)
			assert.Equal(t, tt.want.id, *m.MobileDevices[0].Id)
			assert.Equal(t, tt.want.name, *m.MobileDevices[0].Name)
			require.NotNil(t, *m.MobileDevices[0].Settings.GeoTrackingEnabled)
			assert.Equal(t, tt.want.isGeoTrackEnabled, *m.MobileDevices[0].Settings.GeoTrackingEnabled)
			switch tt.want.hasLocation {
			case true:
				require.NotNil(t, m.MobileDevices[0].Location)
				assert.Equal(t, tt.want.isHome, *m.MobileDevices[0].Location.AtHome)
				assert.Equal(t, tt.want.isStale, *m.MobileDevices[0].Location.Stale)
			case false:
				assert.Nil(t, m.MobileDevices[0].Location)
			}
		})
	}
}
