package controller

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
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
		update poller.Update
		want   action
		err    assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			script: `
function Evaluate(home, zone, devices)
	return zone, 300, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{overlay: true, heating: true},
					reason: "test",
					delay:  5 * time.Minute,
				},
				homeId:   1,
				zoneName: "foo",
				zoneId:   10,
			},
			err: assert.NoError,
		},
		{
			name: "invalid delay",
			script: `
function Evaluate(home, zone, devices)
	return zone, nil, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			err: assert.Error,
		},
		{
			name: "missing Evaluate function",
			script: `
function NotEvaluate(home, zone, devices)
	return zone, 0, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			err: assert.Error,
		},
		{
			name: "missing zone in update",
			script: `
function Evaluate(home, zone, devices)
	return zone, 300, "test"
end
`,
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
			),
			err: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := newZoneRule("foo", strings.NewReader(tt.script), nil, nil)
			require.NoError(t, err)
			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err == nil {
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

func TestZoneRule_LimitOverlay(t *testing.T) {}

func TestZoneRule_AutoAway(t *testing.T) {
	tests := []struct {
		name   string
		update poller.Update
		err    assert.ErrorAssertionFunc
		want   action
	}{
		{
			name: "zone auto, user home -> heating in auto",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{overlay: false, heating: true},
					reason: "one or more users are home: user A",
					delay:  0,
				},
				homeId:   1,
				zoneId:   10,
				zoneName: "zone",
			},
		},
		{
			name: "zone auto, user away -> heating off",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: true, heating: false}, reason: "all users are away", delay: 15 * time.Minute},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
		{
			name: "zone off, user away -> heating off",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: true, heating: false}, reason: "all users are away", delay: 15 * time.Minute},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
		{
			name: "zone off, user home -> zone to auto",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
			),
			err: assert.NoError,
			want: &zoneAction{
				coreAction: coreAction{state: zoneState{overlay: false, heating: true}, reason: "one or more users are home: user A", delay: 0},
				homeId:     1,
				zoneId:     10,
				zoneName:   "zone",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := loadZoneRule(
				"zone",
				RuleConfiguration{
					Script: ScriptConfig{Packaged: "autoaway.lua"},
					Users:  []string{"user A"},
				})
			require.NoError(t, err)

			a, err := r.Evaluate(tt.update)
			tt.err(t, err)
			if err == nil {
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

/*
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

*/
