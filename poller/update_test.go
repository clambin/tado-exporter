package poller_test

import (
	"bytes"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
	"testing"
)

var (
	testUpdate = poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
	}
)

func TestUpdate_GetZoneID(t *testing.T) {
	tests := []struct {
		name string
		zone string
		pass bool
		id   int
	}{
		{
			name: "pass",
			zone: "foo",
			pass: true,
			id:   1,
		},
		{
			name: "fail",
			zone: "snafu",
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoneID, ok := testUpdate.GetZoneID(tt.zone)
			assert.Equal(t, tt.pass, ok)
			if tt.pass {
				assert.Equal(t, tt.id, zoneID)
			}
		})
	}
}

func TestUpdate_GetUserID(t *testing.T) {
	tests := []struct {
		name string
		zone string
		pass bool
		id   int
	}{
		{
			name: "pass",
			zone: "foo",
			pass: true,
			id:   1,
		},
		{
			name: "fail",
			zone: "snafu",
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoneID, ok := testUpdate.GetUserID(tt.zone)
			assert.Equal(t, tt.pass, ok)
			if tt.pass {
				assert.Equal(t, tt.id, zoneID)
			}
		})
	}
}

func TestMobileDevices_LogValue(t *testing.T) {
	devices := poller.MobileDevices{
		10: {
			ID:       10,
			Name:     "home",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: true},
		},
		11: {
			ID:       11,
			Name:     "away",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: false},
		},
		12: {
			ID:       12,
			Name:     "stale",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: true, AtHome: false},
		},
		13: {
			ID:       13,
			Name:     "not geotagged",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: false},
		},
	}

	out := bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(&out))
	logger.Info("devices", "devices", devices)

	assert.Contains(t, out.String(), `devices.device_10.id=10 devices.device_10.name=home devices.device_10.geotracked=true devices.device_10.location.home=true devices.device_10.location.stale=false`)
	assert.Contains(t, out.String(), `devices.device_11.id=11 devices.device_11.name=away devices.device_11.geotracked=true devices.device_11.location.home=false devices.device_11.location.stale=false `)
	assert.Contains(t, out.String(), `devices.device_12.id=12 devices.device_12.name=stale devices.device_12.geotracked=true devices.device_12.location.home=false devices.device_12.location.stale=true`)
	assert.Contains(t, out.String(), `devices.device_13.id=13 devices.device_13.name="not geotagged" devices.device_13.geotracked=false devices.device_13.location.home=false devices.device_13.location.stale=false`)
}
