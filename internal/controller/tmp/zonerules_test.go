package tmp

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestZoneRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update
		want action
		err  assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			script: `
function Evaluate(homeState, zoneState, devices, args)
	return { Heating = zoneState.Heating, Manual = zoneState.Manual }, 300, "test"
end
`,
			update: update{map[string]zoneInfo{"foo": {10, zoneState{true, false}}}, nil, 1, homeState{true, false}},
			want: &zoneAction{
				coreAction: coreAction{zoneState{true, false}, "test", 5 * time.Minute},
				homeId:     1,
				zoneId:     10,
				zoneName:   "foo",
			},
			err: assert.NoError,
		},
		{
			name: "invalid state",
			script: `
function Evaluate(homeState, zoneState, devices, args)
	return "foo", 300, "test"
end
`,
			update: update{map[string]zoneInfo{"foo": {10, zoneState{true, false}}}, nil, 1, homeState{true, false}},
			err:    assert.Error,
		},
		{
			name: "invalid delay",
			script: `
function Evaluate(homeState, zoneState, devices, args)
	return zoneState, nil, "test"
end
`,
			update: update{map[string]zoneInfo{"foo": {10, zoneState{true, false}}}, nil, 1, homeState{true, false}},
			err:    assert.Error,
		},
		{
			name: "missing Evaluate function",
			script: `
function NotEvaluate(homeState, zoneState, devices, args)
	return zoneState, 300, "test"
end
`,
			update: update{map[string]zoneInfo{"foo": {10, zoneState{true, false}}}, nil, 1, homeState{true, false}},
			err:    assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newZoneRule("foo", strings.NewReader(tt.script), nil, nil)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, a)
		})
	}
}

func TestZoneRule_Evaluate_AutoAway(t *testing.T) {
	tests := []struct {
		name string
		update
		want action
		err  assert.ErrorAssertionFunc
	}{
		{
			name: "zone is heating, user is home: no action",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{true, false}}},
				devices:    devices{{Name: "user", Home: true}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			want: &zoneAction{coreAction{zoneState{true, true}, "one or more users are home: user", 0}, "foo", 1, 10},
			err:  assert.NoError,
		},
		{
			name: "zone is heating, user is away: switch off heating",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{true, false}}},
				devices:    devices{{Name: "user", Home: false}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			want: &zoneAction{coreAction{zoneState{false, true}, "all users are away", 15 * time.Minute}, "foo", 1, 10},
			err:  assert.NoError,
		},
		{
			name: "zone is off, user is home: switch on heating",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{false, true}}},
				devices:    devices{{Name: "user", Home: true}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			want: &zoneAction{coreAction{zoneState{true, true}, "one or more users are home: user", 0}, "foo", 1, 10},
			err:  assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := loadZoneRule(RuleConfiguration{
				Name:   "foo",
				Script: ScriptConfig{Packaged: "autoaway.lua"},
				Users:  []string{"user"},
			})
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, a)
		})
	}
}

func TestZoneRule_Evaluate_LimitOverlay(t *testing.T) {
	tests := []struct {
		name string
		update
		want action
		err  assert.ErrorAssertionFunc
	}{
		{
			name: "zone has no overlay: no action",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{true, false}}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			want: &zoneAction{coreAction{zoneState{true, false}, "no manual setting detected", 0}, "foo", 1, 10},
			err:  assert.NoError,
		},
		{
			name: "zone has overlay: remove action",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{true, true}}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			want: &zoneAction{coreAction{zoneState{true, false}, "manual setting detected", time.Hour}, "foo", 1, 10},
			err:  assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := loadZoneRule(RuleConfiguration{
				Name:   "foo",
				Script: ScriptConfig{Packaged: "limitoverlay.lua"},
			})
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, a)
		})
	}
}

func TestZoneRule_Evaluate_NightTime(t *testing.T) {
	tests := []struct {
		name string
		update
		offset int
		want   action
		err    assert.ErrorAssertionFunc
	}{
		{
			name: "zone has no overlay: no action",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{true, false}}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			want: &zoneAction{coreAction{zoneState{true, false}, "no manual setting detected", 0}, "foo", 1, 10},
			err:  assert.NoError,
		},
		{
			name: "zone in overlay: set to auto with delay",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{true, true}}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			offset: 1,
			want:   &zoneAction{coreAction{zoneState{true, false}, "manual setting detected", 59 * time.Minute}, "foo", 1, 10},
			err:    assert.NoError,
		},
		{
			name: "zone in overlay during nighttime: set to auto without delay",
			update: update{
				ZoneStates: map[string]zoneInfo{"foo": {10, zoneState{true, true}}},
				HomeId:     1,
				homeState:  homeState{true, false},
			},
			offset: -1,
			want:   &zoneAction{coreAction{zoneState{true, false}, "manual setting detected", 0}, "foo", 1, 10},
			err:    assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().Local()
			nowHour := now.Hour()
			nowMin := now.Minute()
			r, err := loadZoneRule(RuleConfiguration{
				Name:   "foo",
				Script: ScriptConfig{Packaged: "nighttime.lua"},
				Users:  []string{"user"},
				Args: Args{
					"StartHour": nowHour + tt.offset, "StartMin": nowMin,
					"EndHour": nowHour + tt.offset + 2, "EndMin": nowMin,
				},
			})
			require.NoError(t, err)

			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err != nil {
				return
			}
			if a.(*zoneAction).delay > 0 {
				a.(*zoneAction).delay = a.(*zoneAction).delay.Truncate(time.Minute)
			}
			assert.Equal(t, tt.want, a)
		})
	}
}
