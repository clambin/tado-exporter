package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
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

	type want struct {
		wantError assert.ErrorAssertionFunc
		action    bool
		delay     time.Duration
		reason    string
	}
	testCases := []struct {
		name      string
		config    configuration.ZoneConfiguration
		update    poller.Update
		timestamp time.Time
		want
	}{
		{
			name:   "limitOverlay before nightTime",
			config: cfg,
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
				Home:     true,
			},
			timestamp: time.Date(2023, time.December, 31, 11, 0, 0, 0, time.Local),
			want: want{
				wantError: assert.NoError,
				action:    true,
				delay:     30 * time.Minute,
				reason:    "manual temp setting detected",
			},
		},
		{
			name:   "limitOverlay after nightTime",
			config: cfg,
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
				Home:     true,
			},
			timestamp: time.Date(2023, time.December, 31, 23, 15, 0, 0, time.Local),
			want: want{
				wantError: assert.NoError,
				action:    true,
				delay:     15 * time.Minute,
				reason:    "manual temp setting detected",
			},
		},
		{
			name:   "limitOverlay vs autoAway",
			config: cfg,
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(false))},
				Home:     true,
			},
			timestamp: time.Date(2023, time.December, 31, 11, 15, 0, 0, time.Local),
			want: want{
				wantError: assert.NoError,
				action:    true,
				delay:     30 * time.Minute,
				reason:    "A is away",
			},
		},
		{
			name:   "away",
			config: cfg,
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(false))},
				Home:     false,
			},
			timestamp: time.Date(2023, time.December, 31, 11, 15, 0, 0, time.Local),
			want: want{
				wantError: assert.NoError,
				action:    true,
				reason:    "home in AWAY mode, manual temp setting detected",
			},
		},
		{
			name:   "no action",
			config: cfg,
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
				Home:     true,
			},
			timestamp: time.Date(2023, time.December, 31, 23, 15, 0, 0, time.Local),
			want: want{
				wantError: assert.NoError,
				action:    false,
				delay:     0,
				//reason:    "A is home, home in HOME mode, no manual temp setting detected",
				reason: "A is home, no manual temp setting detected",
			},
		},
		{
			name: "invalid config (zone)",
			config: configuration.ZoneConfiguration{
				Name: "invalid room",
				Rules: configuration.ZoneRuleConfiguration{
					AutoAway:     configuration.AutoAwayConfiguration{Users: []string{"A"}, Delay: 30 * time.Minute},
					LimitOverlay: configuration.LimitOverlayConfiguration{Delay: 30 * time.Minute},
					NightTime:    configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Active: true}},
				},
			},
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
				Home:     true,
			},
			want: want{
				wantError: assert.Error,
			},
		},
		{
			name: "invalid config (user)",
			config: configuration.ZoneConfiguration{
				Name: "room",
				Rules: configuration.ZoneRuleConfiguration{
					AutoAway:     configuration.AutoAwayConfiguration{Users: []string{"B"}, Delay: 30 * time.Minute},
					LimitOverlay: configuration.LimitOverlayConfiguration{Delay: 30 * time.Minute},
					NightTime:    configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Active: true}},
				},
			},
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
				Home:     true,
			},
			want: want{
				wantError: assert.Error,
			},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := LoadZoneRules(tt.config, tt.update, slog.Default())
			tt.want.wantError(t, err)

			if err != nil {
				return
			}

			require.Len(t, r, 4)
			r[3].(*NightTimeRule).GetCurrentTime = func() time.Time { return tt.timestamp }

			e, err := r.Evaluate(tt.update)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.action, e.IsAction())
			assert.Equal(t, tt.want.delay, e.Delay)
			assert.Equal(t, tt.want.reason, e.Reason)
		})
	}
}

func TestRules_ZoneRules_Empty(t *testing.T) {
	cfg := configuration.ZoneConfiguration{
		Name: "room",
	}
	update := poller.Update{Zones: map[int]tado.Zone{
		10: {ID: 10, Name: "room"},
	}}

	r, err := LoadZoneRules(cfg, update, slog.Default())
	require.NoError(t, err)

	assert.Len(t, r, 0)
}
