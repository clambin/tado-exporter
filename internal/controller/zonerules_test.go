package controller

import (
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/controller/zonerules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type zoneWant struct {
	zoneState
	delay  time.Duration
	reason string
	err    assert.ErrorAssertionFunc
}

func TestZoneRules(t *testing.T) {
	type want struct {
		zoneState
		delay  time.Duration
		reason string
		err    assert.ErrorAssertionFunc
	}
	tests := []struct {
		name string
		zoneRules
		update
		want
	}{
		{
			name: "no rules",
			want: want{"", 0, "no rules found", assert.Error},
		},
		{
			name: "single rule",
			zoneRules: zoneRules{
				zoneName: "foo",
				rules: []evaluator{
					fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
				},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with same desired zone state: pick the first one",
			zoneRules: zoneRules{
				zoneName: "foo",
				rules: []evaluator{
					fakeZoneEvaluator{ZoneStateAuto, time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateAuto, time.Hour, "manual setting detected", nil},
				},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices: nil},
			want:   want{ZoneStateAuto, time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states: pick the first one",
			zoneRules: zoneRules{
				zoneName: "foo",
				rules: []evaluator{
					fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
				},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states, including `no change`: pick the first non-matching",
			zoneRules: zoneRules{
				zoneName: "foo",
				rules: []evaluator{
					fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
					fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
				},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different 'no-change' actions: join the reasons",
			zoneRules: zoneRules{
				zoneName: "foo",
				rules: []evaluator{
					fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
					fakeZoneEvaluator{ZoneStateAuto, 0, "users are home", nil},
					fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
				},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 0, "no manual setting detected, users are home", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := tt.zoneRules.Evaluate(tt.update)
			tt.want.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want.zoneState, zoneState(a.GetState()))
			assert.Equal(t, tt.want.delay, a.GetDelay())
			assert.Equal(t, tt.want.reason, a.GetReason())
		})
	}
}

func TestZoneRules_AutoAway_vs_LimitOverlay(t *testing.T) {
	autoAwayCfg := RuleConfiguration{"", ScriptConfig{Packaged: "autoaway.lua"}, []string{"user"}}
	limitOverlayCfg := RuleConfiguration{"", ScriptConfig{Packaged: "limitoverlay.lua"}, nil}

	tests := []struct {
		name   string
		rules  []RuleConfiguration
		update update
		want   zoneAction
	}{
		{
			name:  "user is home: no action",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {zoneState: ZoneStateAuto, ZoneId: 1}},
				devices:    []device{{"user", true}},
			},
			want: zoneAction{ZoneStateAuto, 0, "no manual setting detected, one or more users are home: user", 1, 1, "zone"},
		},
		{
			name:  "user is not home: switch heating is off",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {ZoneStateAuto, 1}},
				devices:    []device{{"user", false}},
			},
			want: zoneAction{ZoneStateOff, 15 * time.Minute, "all users are away", 1, 1, "zone"},
		},
		{
			name:  "user is not home, heating is off: no action",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {ZoneStateOff, 1}},
				devices:    []device{{"user", false}},
			},
			want: zoneAction{ZoneStateOff, 15 * time.Minute, "all users are away", 1, 1, "zone"},
		},
		{
			name:  "user is home, heating is off: move heating to auto mode",
			rules: []RuleConfiguration{limitOverlayCfg, autoAwayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {ZoneStateOff, 1}},
				devices:    []device{{"user", true}},
			},
			want: zoneAction{ZoneStateAuto, 0, "one or more users are home: user", 1, 1, "zone"},
		},
		{
			name:  "user is home, zone in manual mode: schedule auto mode",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {zoneState: ZoneStateManual, ZoneId: 1}},
				devices:    []device{{"user", true}},
			},
			want: zoneAction{ZoneStateAuto, time.Hour, "manual setting detected", 1, 1, "zone"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zr, err := loadZoneRules("zone", tt.rules)
			require.NoError(t, err)

			a, err := zr.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.want, a)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestZoneRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update
		zoneWant
	}{
		{
			name: "success",
			script: `
function Evaluate(home, zone, devices)
	return zone, 300, "test"
end
`,
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateOff}}, devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{ZoneStateOff, 5 * time.Minute, "test", assert.NoError},
		},
		{
			name: "invalid delay",
			script: `
function Evaluate(home, zone, devices)
	return zone, nil, "test"
end
`,
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateOff}}, devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{"", 0, "", assert.Error},
		},
		{
			name: "missing Evaluate function",
			script: `
function NotEvaluate(home, zone, devices)
	return zone, 0, "test"
end
`,
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateOff}}, devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{"", 0, "", assert.Error},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newZoneRule("foo", strings.NewReader(tt.script), nil)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			assert.Equal(t, tt.zoneWant.zoneState, zoneState(a.GetState()))
			assert.Equal(t, tt.zoneWant.delay, a.GetDelay())
			assert.Equal(t, tt.zoneWant.reason, a.GetReason())
			tt.zoneWant.err(t, err)
		})
	}
}

func TestZoneRule_UseCases(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update
		zoneWant
	}{
		{
			name:     "limitOverlay - auto",
			script:   "limitoverlay.lua",
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name:     "limitOverlay - manual",
			script:   "limitoverlay.lua",
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateOff}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, time.Hour, "manual setting detected", assert.NoError},
		},
		{
			name:     "autoAway - home",
			script:   "autoaway.lua",
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "one or more users are home: user", assert.NoError},
		},
		{
			name:     "autoAway - away",
			script:   "autoaway.lua",
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices{{Name: "user", Home: false}}},
			zoneWant: zoneWant{ZoneStateOff, 15 * time.Minute, "all users are away", assert.NoError},
		},
		{
			name:     "autoAway - no valid users",
			script:   "autoaway.lua",
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices{{Name: "bar", Home: false}}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "no devices found", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := zonerules.FS.Open(tt.script)
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })
			r, err := newZoneRule("foo", f, []string{"user"})
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			assert.Equal(t, tt.zoneWant.zoneState, zoneState(a.GetState()))
			assert.Equal(t, tt.zoneWant.delay, a.GetDelay())
			assert.Equal(t, tt.zoneWant.reason, a.GetReason())
			assert.NoError(t, err)
		})
	}
}

func TestZoneRule_UseCases_Nighttime(t *testing.T) {
	tests := []struct {
		name string
		now  time.Time
		update
		zoneWant
	}{
		{
			name:     "no manual setting",
			now:      time.Date(2024, time.November, 26, 12, 0, 0, 0, time.Local),
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name:     "nightTime",
			now:      time.Date(2024, time.November, 26, 1, 0, 0, 0, time.Local),
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "manual setting detected", assert.NoError},
		},
		{
			name:     "daytime",
			now:      time.Date(2024, time.November, 26, 12, 0, 0, 0, time.Local),
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 12 * time.Hour, "manual setting detected", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := zonerules.FS.Open("nighttime.lua")
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })
			r, err := newZoneRule("foo", f, nil)
			require.NoError(t, err)

			// re-register functions with custom "now" function
			luart.Register(r.State, func() time.Time { return tt.now })

			a, err := r.Evaluate(tt.update)
			assert.Equal(t, tt.zoneWant.zoneState, zoneState(a.GetState()))
			assert.Equal(t, tt.zoneWant.delay, a.GetDelay())
			assert.Equal(t, tt.zoneWant.reason, a.GetReason())
			assert.NoError(t, err)
		})
	}
}

func BenchmarkZoneEvaluator(b *testing.B) {
	f, err := zonerules.FS.Open("nighttime.lua")
	require.NoError(b, err)
	b.Cleanup(func() { _ = f.Close() })
	r, err := newZoneRule("foo", f, []string{"user"})
	require.NoError(b, err)
	u := update{
		homeState:  HomeStateAuto,
		ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}},
		devices:    devices{},
	}
	b.ResetTimer()
	for range b.N {
		if _, err := r.Evaluate(u); err != nil {
			b.Fatal(err)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ evaluator = fakeZoneEvaluator{}

type fakeZoneEvaluator struct {
	zoneState
	delay  time.Duration
	reason string
	err    error
}

func (f fakeZoneEvaluator) Evaluate(_ update) (action, error) {
	return zoneAction{zoneState: f.zoneState, delay: f.delay, reason: f.reason}, f.err
}
