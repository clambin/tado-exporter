package poller_test

import (
	"bytes"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
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
		name   string
		zone   string
		wantOK bool
		want   int
	}{
		{
			name:   "pass",
			zone:   "foo",
			wantOK: true,
			want:   1,
		},
		{
			name:   "fail",
			zone:   "snafu",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoneID, ok := testUpdate.GetZoneID(tt.zone)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, zoneID)
		})
	}
}

func TestUpdate_GetUserID(t *testing.T) {
	tests := []struct {
		name   string
		zone   string
		wantOK bool
		want   int
	}{
		{
			name:   "pass",
			zone:   "foo",
			wantOK: true,
			want:   1,
		},
		{
			name:   "fail",
			zone:   "snafu",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoneID, ok := testUpdate.GetUserID(tt.zone)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, zoneID)
		})
	}
}

func TestMobileDevices_LogValue(t *testing.T) {
	devices := poller.MobileDevices{
		10: testutil.MakeMobileDevice(10, "home", testutil.Home(true)),
		11: testutil.MakeMobileDevice(11, "away", testutil.Home(false)),
		12: {
			ID:       12,
			Name:     "stale",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: true, AtHome: false},
		},
		13: testutil.MakeMobileDevice(13, "not geotagged"),
	}

	out := bytes.Buffer{}
	// TODO: filter out timestamp so we can do a straight comparison
	logger := slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{}))
	logger.Info("devices", "devices", devices)

	logEntry := out.String()
	assert.Contains(t, logEntry, ` devices.device_10.id=10 devices.device_10.name=home devices.device_10.geotracked=true devices.device_10.location.home=true devices.device_10.location.stale=false`)
	assert.Contains(t, logEntry, ` devices.device_11.id=11 devices.device_11.name=away devices.device_11.geotracked=true devices.device_11.location.home=false devices.device_11.location.stale=false`)
	assert.Contains(t, logEntry, ` devices.device_12.id=12 devices.device_12.name=stale devices.device_12.geotracked=true devices.device_12.location.home=false devices.device_12.location.stale=true`)
	assert.Contains(t, logEntry, ` devices.device_13.id=13 devices.device_13.name="not geotagged" devices.device_13.geotracked=false devices.device_13.location.home=false devices.device_13.location.stale=false`)
}
