package rules

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestHomeAwayRule_Evaluate(t *testing.T) {
	type want struct {
		err     assert.ErrorAssertionFunc
		action  assert.BoolAssertionFunc
		delay   time.Duration
		reason  string
		overlay bool
	}

	var testCases = []struct {
		name   string
		update poller.Update
		want
	}{
		{
			name: "home in HOME mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 22))},
				Home:     true,
			},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				//reason: "home in HOME mode",
			},
		},
		{
			name: "home in AWAY mode, overlay set",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay())},
				Home:     false,
			},
			want: want{
				err:    assert.NoError,
				action: assert.True,
				reason: "home in AWAY mode, manual temp setting detected",
			},
		},
		{
			name: "home in AWAY mode, zone in auto mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				Home:     false,
			},
			want: want{
				err:     assert.NoError,
				action:  assert.False,
				reason:  "home in AWAY mode, no manual temp setting detected",
				overlay: false,
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := LoadHomeAwayRule(10, "room", tt.update, slog.Default())
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
