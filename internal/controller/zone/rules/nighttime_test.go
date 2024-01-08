package rules

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestNightTimeRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action assert.BoolAssertionFunc
		delay  time.Duration
		reason string
	}

	testCases := []struct {
		name   string
		update poller.Update
		cfg    configuration.NightTimeConfiguration
		now    time.Time
		want
	}{
		{
			name: "zone in auto mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 22))},
				Home:     true,
			},
			cfg: configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Seconds: 0, Active: true}},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "no manual temp setting detected",
			},
		},
		{
			name: "zone in manual mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				Home:     true,
			},
			cfg: configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Seconds: 0, Active: true}},
			now: time.Date(2023, time.December, 31, 22, 30, 0, 0, time.Local),
			want: want{
				err:    assert.NoError,
				action: assert.True,
				delay:  time.Hour,
				reason: "manual temp setting detected",
			},
		},
		{
			name: "zone in manual mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				Home:     true,
			},
			cfg: configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Seconds: 0, Active: true}},
			now: time.Date(2023, time.December, 31, 23, 45, 0, 0, time.Local),
			want: want{
				err:    assert.NoError,
				action: assert.True,
				delay:  23*time.Hour + 45*time.Minute,
				reason: "manual temp setting detected",
			},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, err := LoadNightTime(10, "room", tt.cfg, tt.update, slog.Default())
			tt.err(t, err)
			if err != nil {
				return
			}
			r.GetCurrentTime = func() time.Time { return tt.now }

			e, err := r.Evaluate(tt.update)
			tt.want.err(t, err)
			if err != nil {
				return
			}
			tt.action(t, e.IsAction())
			assert.Equal(t, tt.want.delay, e.Delay)
			assert.Equal(t, tt.want.reason, e.Reason)

			if !e.IsAction() {
				return
			}

			ctx := context.Background()
			c := mocks.NewTadoSetter(t)
			c.EXPECT().DeleteZoneOverlay(ctx, 10).Return(nil).Once()

			assert.NoError(t, e.State.Do(ctx, c))
		})
	}
}
