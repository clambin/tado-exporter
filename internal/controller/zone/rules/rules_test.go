package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRules_ZoneRules(t *testing.T) {
	cfg := configuration.ZoneConfiguration{
		Name: "room",
		Rules: configuration.ZoneRuleConfiguration{
			AutoAway:     configuration.AutoAwayConfiguration{Users: []string{"A"}, Delay: 30 * time.Minute},
			LimitOverlay: configuration.LimitOverlayConfiguration{Delay: 30 * time.Minute},
			NightTime:    configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Active: true}},
		},
	}
	update := poller.Update{
		Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
		UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
		Home:     true,
	}

	type want struct {
		action bool
		delay  time.Duration
		reason string
	}
	testCases := []struct {
		name      string
		update    poller.Update
		timestamp time.Time
		want
	}{
		{
			name: "limitOverlay before nightTime",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
				Home:     true,
			},
			timestamp: time.Date(2023, time.December, 31, 11, 0, 0, 0, time.Local),
			want: want{
				action: true,
				delay:  30 * time.Minute,
				reason: "manual temp setting detected",
			},
		},
		{
			name: "limitOverlay after nightTime",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
				Home:     true,
			},
			timestamp: time.Date(2023, time.December, 31, 23, 15, 0, 0, time.Local),
			want: want{
				action: true,
				delay:  15 * time.Minute,
				reason: "manual temp setting detected",
			},
		},
		{
			name: "limitOverlay vs autoAway",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(false))},
				Home:     true,
			},
			timestamp: time.Date(2023, time.December, 31, 11, 15, 0, 0, time.Local),
			want: want{
				action: true,
				delay:  30 * time.Minute,
				reason: "A is away",
			},
		},
		{
			name:      "no action",
			update:    update,
			timestamp: time.Date(2023, time.December, 31, 23, 15, 0, 0, time.Local),
			want: want{
				action: false,
				delay:  0,
				reason: "A is home, no manual temp setting detected",
			},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := LoadZoneRules(cfg, update)
			require.NoError(t, err)
			require.Len(t, r, 3)
			r[2].(*NightTimeRule).GetCurrentTime = func() time.Time { return tt.timestamp }

			e, err := r.Evaluate(tt.update)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.action, e.IsAction())
			assert.Equal(t, tt.want.delay, e.Delay)
			assert.Equal(t, tt.want.reason, e.Reason)
		})
	}
}
