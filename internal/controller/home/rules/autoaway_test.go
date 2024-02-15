package rules

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestAutoAwayRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action assert.BoolAssertionFunc
		delay  time.Duration
		reason string
		state  action.State
		home   bool
	}

	tests := []struct {
		name   string
		update poller.Update
		want
	}{
		{
			name: "all users are home",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "A", testutil.Home(true)),
					110: testutil.MakeMobileDevice(110, "B", testutil.Home(true)),
				},
				Home: true,
			},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "A, B are home",
			},
		},
		{
			name: "one user is home",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "A", testutil.Home(true)),
					110: testutil.MakeMobileDevice(110, "B", testutil.Home(false)),
				},
				Home: true,
			},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "A is home",
			},
		},
		{
			name: "all users go away",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "A", testutil.Home(false)),
					110: testutil.MakeMobileDevice(110, "B", testutil.Home(false)),
				},
				Home: true,
			},
			want: want{
				err:    assert.NoError,
				action: assert.True,
				delay:  time.Hour,
				reason: "A, B are away",
				state:  State{mode: action.HomeInAwayMode},
				home:   false,
			},
		},
		{
			name: "all users are away",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "A", testutil.Home(false)),
					110: testutil.MakeMobileDevice(110, "B", testutil.Home(false)),
				},
				Home: false,
			},
			want: want{
				err:    assert.NoError,
				action: assert.False,
				reason: "A, B are away",
			},
		},
		{
			name: "user comes home",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "A", testutil.Home(true)),
					110: testutil.MakeMobileDevice(110, "B", testutil.Home(false)),
				},
				Home: false,
			},
			want: want{
				err:    assert.NoError,
				action: assert.True,
				reason: "A is home",
				state:  State{mode: action.HomeInHomeMode},
				home:   true,
			},
		},
	}

	cfg := configuration.AutoAwayConfiguration{
		Users: []string{"A", "B"},
		Delay: time.Hour,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, err := LoadAutoAwayRule(cfg, tt.update, slog.Default())
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

			assert.True(t, e.State.IsEqual(tt.state))
			ctx := context.Background()
			c := mocks.NewTadoSetter(t)
			c.EXPECT().SetHomeState(ctx, tt.want.home).Return(nil).Once()

			assert.NoError(t, e.State.Do(ctx, c))
		})
	}
}

func TestAutoAwayRule_Evaluate_InvalidConfig(t *testing.T) {
	cfg := configuration.AutoAwayConfiguration{
		Users: []string{"A", "B"},
		Delay: time.Hour,
	}
	update := poller.Update{UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "A"}}}
	_, err := LoadAutoAwayRule(cfg, update, slog.Default())
	assert.Error(t, err)
}
