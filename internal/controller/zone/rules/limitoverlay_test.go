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

func TestLimitOverlayRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action assert.BoolAssertionFunc
		delay  time.Duration
		reason string
	}

	tests := []struct {
		name   string
		update poller.Update
		cfg    configuration.LimitOverlayConfiguration
		want
	}{
		{
			name: "zone in auto mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 22))},
				Home:     true,
			},
			cfg: configuration.LimitOverlayConfiguration{Delay: time.Hour},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "no manual temp setting detected",
			},
		},
		{
			name: "zone in manual mode (heating)",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
				Home:     true,
			},
			cfg: configuration.LimitOverlayConfiguration{Delay: time.Hour},
			want: want{
				err:    assert.NoError,
				action: assert.True,
				delay:  time.Hour,
				reason: "manual temp setting detected",
			},
		},
		{
			name: "zone in manual mode (off)",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay())},
				Home:     true,
			},
			cfg: configuration.LimitOverlayConfiguration{Delay: time.Hour},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "no manual temp setting detected",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, err := LoadLimitOverlay(10, "room", tt.cfg, tt.update, slog.Default())
			tt.err(t, err)
			if err != nil {
				return
			}
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
