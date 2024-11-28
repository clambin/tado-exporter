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
			update:   update{HomeStateHome, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateOff}}, devices{{Name: "user", Home: true}}},
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
		{
			name: "missing zone in update",
			script: `
function Evaluate(home, zone, devices)
	return zone, 300, "test"
end
`,
			update:   update{HomeStateAuto, 1, nil, nil},
			zoneWant: zoneWant{"", 0, "", assert.Error},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newZoneRule("foo", strings.NewReader(tt.script), nil, nil)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.zoneWant.err(t, err)
			if err == nil {
				assert.Equal(t, tt.zoneWant.zoneState, zoneState(a.GetState()))
				assert.Equal(t, tt.zoneWant.delay, a.GetDelay())
				assert.Equal(t, tt.zoneWant.reason, a.GetReason())
			}
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
			r, err := newZoneRule("foo", f, []string{"user"}, nil)
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
		args Args
		update
		zoneWant
	}{
		{
			name:     "no manual setting",
			now:      time.Date(2024, time.November, 26, 12, 0, 0, 0, time.Local),
			args:     Args{"StartHour": 23, "StartMin": 30, "EndHour": 6, "EndMin": 0},
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices{{Name: "user", Home: true}}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name:     "nightTime",
			now:      time.Date(2024, time.November, 26, 23, 45, 0, 0, time.Local),
			args:     Args{"StartHour": 23, "StartMin": 30, "EndHour": 6, "EndMin": 0},
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "manual setting detected", assert.NoError},
		},
		{
			name:     "daytime",
			now:      time.Date(2024, time.November, 26, 11, 30, 0, 0, time.Local),
			args:     Args{"StartHour": 23, "StartMin": 30, "EndHour": 6, "EndMin": 0},
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 12 * time.Hour, "manual setting detected", assert.NoError},
		},
		{
			name:     "nightTime - default args",
			now:      time.Date(2024, time.November, 26, 23, 45, 0, 0, time.Local),
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 0, "manual setting detected", assert.NoError},
		},
		{
			name:     "daytime - default args",
			now:      time.Date(2024, time.November, 26, 11, 30, 0, 0, time.Local),
			update:   update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices{}},
			zoneWant: zoneWant{ZoneStateAuto, 12 * time.Hour, "manual setting detected", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := zonerules.FS.Open("nighttime.lua")
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })
			r, err := newZoneRule("foo", f, nil, tt.args)
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

func TestUnit(t *testing.T) {
	f, err := zonerules.FS.Open("nighttime.lua")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	args := map[string]any{
		"StartHour": 23,
		"StartMin":  30,
		"EndHour":   6,
		"EndMin":    0,
	}
	args = nil
	r, err := newZoneRule("foo", f, nil, args)
	require.NoError(t, err)

	// re-register functions with custom "now" function
	now := time.Date(2024, time.November, 26, 1, 0, 0, 0, time.Local)
	luart.Register(r.State, func() time.Time { return now })

	u := update{HomeStateAuto, 1, map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices{}}
	a, err := r.Evaluate(u)
	require.NoError(t, err)
	t.Log(a.GetDelay())
}

func BenchmarkZoneEvaluator(b *testing.B) {
	f, err := zonerules.FS.Open("nighttime.lua")
	require.NoError(b, err)
	b.Cleanup(func() { _ = f.Close() })
	args := map[string]any{
		"StartHour": 23,
		"StartMin":  30,
		"EndHour":   6,
		"EndMin":    0,
	}
	r, err := newZoneRule("foo", f, []string{"user"}, args)
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
	return &zoneAction{zoneState: f.zoneState, delay: f.delay, reason: f.reason}, f.err
}
