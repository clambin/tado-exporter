package rules

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/luart"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLoadZoneRules(t *testing.T) {
	r, err := LoadZoneRules("foo", []RuleConfiguration{
		{Script: ScriptConfig{Packaged: "limitoverlay.lua"}},
		{Script: ScriptConfig{Packaged: "autoaway.lua"}, Users: []string{"user A"}},
		{Script: ScriptConfig{Packaged: "nighttime.lua"}},
	})
	require.NoError(t, err)
	require.Equal(t, 3, r.Count())
	for _, rule := range r.rules {
		_, ok := rule.(zoneRule)
		assert.True(t, ok)
	}

}

func TestZoneRule_Evaluate(t *testing.T) {
	tests := []struct {
		name   string
		script string
		update poller.Update
		want   Action
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
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			want: &zoneAction{
				reason:    "test",
				zoneName:  "foo",
				delay:     5 * time.Minute,
				HomeId:    1,
				ZoneId:    10,
				ZoneState: ZoneState{true, true},
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
				testutils.WithZone(10, "foo", tado.PowerON, 0, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			err: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := LoadZoneRule("foo", RuleConfiguration{Name: tt.name, Script: ScriptConfig{Text: tt.script}})
			require.NoError(t, err)

			s, err := GetZoneState("foo")(tt.update)
			require.NoError(t, err)

			a, err := r.Evaluate(s)
			tt.err(t, err)
			if err == nil {
				assert.Equal(t, tt.want, a)
			}
		})
	}
}

type zoneTest struct {
	name     string
	update   poller.Update
	zoneName string
	now      time.Time
	want     Action
	err      assert.ErrorAssertionFunc
}

func testZoneRule(t *testing.T, script string, tt zoneTest) {
	t.Helper()
	r, err := LoadZoneRule(tt.zoneName, RuleConfiguration{Name: tt.name, Script: ScriptConfig{Packaged: script}})
	require.NoError(t, err)
	if !tt.now.IsZero() {
		luart.Register(r.(zoneRule).luaScript.State, func() time.Time { return tt.now })
	}
	s, err := GetZoneState(tt.zoneName)(tt.update)
	require.NoError(t, err)

	a, err := r.Evaluate(s)
	tt.err(t, err)
	if err == nil {
		assert.Equal(t, tt.want, a)
	}
}

func TestZoneRule_Evaluate_LimitOverlay(t *testing.T) {
	tests := []zoneTest{
		{
			name: "zone auto -> no action",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"no manual setting detected", "zone", 0, 1, 10, ZoneState{false, true}},
		},
		{
			name: "zone manual -> delete overlay",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"manual setting detected", "zone", time.Hour, 1, 10, ZoneState{false, true}},
		},
		{
			name: "zone off -> no action",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"heating is off", "zone", 0, 1, 10, ZoneState{true, false}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testZoneRule(t, "limitoverlay.lua", tt)
		})
	}
}

func TestZoneRule_Evaluate_AutoAway(t *testing.T) {
	tests := []zoneTest{
		{
			name: "zone auto, user home -> heating in auto",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"one or more users are home: user A", "zone", 0, 1, 10, ZoneState{false, true}},
		},
		{
			name: "zone auto, user away -> heating off",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"all users are away: user A", "zone", 15 * time.Minute, 1, 10, ZoneState{true, false}},
		},
		{
			name: "zone off, user away -> heating off",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"all users are away: user A", "zone", 0, 1, 10, ZoneState{true, false}},
		},
		{
			name: "zone off, user home -> zone to auto",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"one or more users are home: user A", "zone", 0, 1, 10, ZoneState{false, true}},
		},
	}

	for _, tt := range tests {
		testZoneRule(t, "autoaway.lua", tt)
	}
}

func TestZoneRule_Evaluate_NightTime(t *testing.T) {
	tests := []zoneTest{
		{
			name: "zone auto -> no action",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
			),
			zoneName: "zone",
			err:      assert.NoError,
			want:     &zoneAction{"no manual setting detected", "zone", 0, 1, 10, ZoneState{false, true}},
		},
		{
			name: "zone in manual mode, before range -> schedule moving to auto mode",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			zoneName: "zone",
			now:      time.Date(2024, time.December, 3, 18, 30, 0, 0, time.Local),
			err:      assert.NoError,
			want:     &zoneAction{"manual setting detected", "zone", 5 * time.Hour, 1, 10, ZoneState{false, true}},
		},
		{
			name: "zone in manual mode, during range, before midnight -> immediately move to auto mode",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			zoneName: "zone",
			now:      time.Date(2024, time.December, 3, 23, 30, 0, 0, time.Local),
			err:      assert.NoError,
			want:     &zoneAction{"manual setting detected", "zone", 0, 1, 10, ZoneState{false, true}},
		},
		{
			name: "zone in manual mode, during range, after midnight -> immediately move to auto mode",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			zoneName: "zone",
			now:      time.Date(2024, time.December, 4, 1, 0, 0, 0, time.Local),
			err:      assert.NoError,
			want:     &zoneAction{"manual setting detected", "zone", 0, 1, 10, ZoneState{false, true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testZoneRule(t, "nighttime.lua", tt)
		})
	}
}
