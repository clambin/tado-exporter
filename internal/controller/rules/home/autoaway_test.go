package home

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/rules/evaluate/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayRule_Evaluate(t *testing.T) {
	type want struct {
		err    assert.ErrorAssertionFunc
		action assert.BoolAssertionFunc
		delay  time.Duration
		reason string
		home   bool
	}

	var testCases = []struct {
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
				home:   true,
			},
		},
	}

	cfg := configuration.AutoAwayConfiguration{
		Users: []string{"A", "B"},
		Delay: time.Hour,
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, err := LoadAutoAwayRule(cfg, tt.update)
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
			c.EXPECT().SetHomeState(ctx, tt.want.home).Return(nil).Maybe()

			assert.NoError(t, e.Do(ctx, c))
		})
	}
}

func TestAutoAwayRule_Evaluate_InvalidConfig(t *testing.T) {
	cfg := configuration.AutoAwayConfiguration{
		Users: []string{"A", "B"},
		Delay: time.Hour,
	}
	update := poller.Update{UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "A"}}}
	_, err := LoadAutoAwayRule(cfg, update)
	assert.Error(t, err)
}
