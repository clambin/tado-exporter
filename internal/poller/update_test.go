package poller_test

import (
	"bytes"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"strconv"
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
	logger := slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		// Remove time from the output for predictable test output.
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}))
	logger.Info("devices", "devices", devices)

	logEntry := out.String()
	assert.Contains(t, logEntry, ` devices.device_10.id=10 devices.device_10.name=home devices.device_10.geotracked=true devices.device_10.location.home=true devices.device_10.location.stale=false`)
	assert.Contains(t, logEntry, ` devices.device_11.id=11 devices.device_11.name=away devices.device_11.geotracked=true devices.device_11.location.home=false devices.device_11.location.stale=false`)
	assert.Contains(t, logEntry, ` devices.device_12.id=12 devices.device_12.name=stale devices.device_12.geotracked=true devices.device_12.location.home=false devices.device_12.location.stale=true`)
	assert.Contains(t, logEntry, ` devices.device_13.id=13 devices.device_13.name="not geotagged" devices.device_13.geotracked=false devices.device_13.location.home=false devices.device_13.location.stale=false`)
}

func BenchmarkUpdate_copy(b *testing.B) {
	u := makeBigUpdate()
	ch := make(chan poller.Update, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- u
	}
	for i := 0; i < b.N; i++ {
		<-ch
	}
}

func BenchmarkUpdate_pointer(b *testing.B) {
	u := makeBigUpdate()
	ch := make(chan *poller.Update, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- &u
	}
	for i := 0; i < b.N; i++ {
		<-ch
	}
}

func makeBigUpdate() poller.Update {
	const userCount = 10
	const zoneCount = 20
	u := poller.Update{
		UserInfo: make(poller.MobileDevices),
		Zones:    make(map[int]tado.Zone),
		ZoneInfo: make(map[int]tado.ZoneInfo),
	}
	for i := 0; i < userCount; i++ {
		u.UserInfo[i] = tado.MobileDevice{Name: strconv.Itoa(i), ID: i}
	}
	for i := 0; i < zoneCount; i++ {
		u.Zones[i] = tado.Zone{Name: strconv.Itoa(i), ID: i}
		u.ZoneInfo[i] = tado.ZoneInfo{}
	}

	return u
}
