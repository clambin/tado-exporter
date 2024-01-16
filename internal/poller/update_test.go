package poller_test

import (
	"bytes"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/testutil"
	"github.com/clambin/tado-exporter/internal/poller"
	tadoTestutil "github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"slices"
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

func TestIsHome_String(t *testing.T) {
	u := poller.Update{Home: true}
	assert.Equal(t, "HOME", u.Home.String())
	u.Home = false
	assert.Equal(t, "AWAY", u.Home.String())
}

func TestMobileDevices_LogValue(t *testing.T) {
	devices := poller.MobileDevices{
		10: tadoTestutil.MakeMobileDevice(10, "home", tadoTestutil.Home(true)),
		11: tadoTestutil.MakeMobileDevice(11, "away", tadoTestutil.Home(false)),
		12: {
			ID:       12,
			Name:     "stale",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: true, AtHome: false},
		},
		13: tadoTestutil.MakeMobileDevice(13, "not geotagged"),
	}

	var out bytes.Buffer
	logger := testutil.NewBufferLogger(&out)
	logger.Info("devices", "devices", devices)

	logEntry := out.String()
	assert.Equal(t, `level=INFO msg=devices devices.device_10.id=10 devices.device_10.name=home devices.device_10.geotracked=true devices.device_10.location.home=true devices.device_10.location.stale=false devices.device_11.id=11 devices.device_11.name=away devices.device_11.geotracked=true devices.device_11.location.home=false devices.device_11.location.stale=false devices.device_12.id=12 devices.device_12.name=stale devices.device_12.geotracked=true devices.device_12.location.home=false devices.device_12.location.stale=true devices.device_13.id=13 devices.device_13.name="not geotagged" devices.device_13.geotracked=false
`, logEntry)
}

func TestUpdate_GetDeviceStatus(t *testing.T) {
	testCases := []struct {
		name     string
		ids      []int
		wantHome []string
		wantAway []string
	}{
		{
			name:     "no ids",
			wantHome: nil,
			wantAway: []string{"bar", "foo"},
		},
		{
			name:     "ids",
			ids:      []int{1, 2},
			wantHome: nil,
			wantAway: []string{"bar", "foo"},
		},
		{
			name:     "ids",
			ids:      []int{1},
			wantHome: nil,
			wantAway: []string{"foo"},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			home, away := testUpdate.GetDeviceStatus(tt.ids...)
			slices.Sort(home)
			slices.Sort(away)
			assert.Equal(t, tt.wantHome, home)
			assert.Equal(t, tt.wantAway, away)
		})
	}
}
