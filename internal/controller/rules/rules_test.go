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

func TestRules_zoneRules(t *testing.T) {
	r, err := LoadZoneRules("zone", []RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "autoaway"}, Users: []string{"user"}},
		{Name: "limitOverlay", Script: ScriptConfig{Packaged: "limitoverlay"}},
		{Name: "nightTime", Script: ScriptConfig{Packaged: "nighttime"}, Args: map[string]any{"StartHour": 23, "StartMin": 0, "EndHour": 6, "EndMin": 0}},
	})
	require.NoError(t, err)
	require.Len(t, r.rules, 3)
	zr, ok := r.rules[2].(zoneRule)
	require.True(t, ok)
	luart.LoadTadoModule(func() time.Time {
		return time.Date(2024, time.December, 6, 13, 0, 0, 0, time.Local)
	})(zr.luaScript.State)

	tests := []struct {
		name            string
		update          poller.Update
		wantReason      string
		wantDescription string
	}{
		{
			name: "no action",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 19),
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, false)),
			),
			wantReason:      "no manual setting detected, one or more users are home: user",
			wantDescription: "*zone*: switching heating to auto mode in 0s",
		},
		{
			name: "user away",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 19),
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(false, false)),
			),
			wantReason:      "all users are away: user",
			wantDescription: "*zone*: switching heating off in 15m0s",
		},
		{
			name: "zone in manual mode",
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 19, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, false)),
			),
			wantReason:      "manual setting detected",
			wantDescription: "*zone*: switching heating to auto mode in 1h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := r.GetState(tt.update)
			require.NoError(t, err)

			a, err := r.Evaluate(state)
			require.NoError(t, err)
			assert.Equal(t, tt.wantReason, a.Reason())
			assert.Equal(t, tt.wantDescription, a.Description(true))
		})
	}
}

// Current:
// BenchmarkRules_Evaluate/action-16         	  137366	      8526 ns/op	    6760 B/op	     119 allocs/op
// BenchmarkRules_Evaluate/no_action-16      	  170823	      6967 ns/op	    6336 B/op	     111 allocs/op
func BenchmarkRules_Evaluate(b *testing.B) {
	r, err := LoadZoneRules("zone", []RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "autoaway"}, Users: []string{"user"}},
		{Name: "limitOverlay", Script: ScriptConfig{Packaged: "limitoverlay"}},
		{Name: "nightTime", Script: ScriptConfig{Packaged: "nighttime"}, Args: map[string]any{"StartHour": 23, "StartMin": 0, "EndHour": 6, "EndMin": 0}},
	})
	require.NoError(b, err)
	var a Action
	b.ResetTimer()
	b.Run("action", func(b *testing.B) {
		b.ReportAllocs()
		s := State{ZoneState: ZoneState{true, true}}
		for b.Loop() {
			a, err = r.Evaluate(s)
			if err != nil {
				b.Fatal(err)
			}
			if !a.IsState(State{ZoneState: ZoneState{false, true}}) {
				b.Fatal("unexpected result")
			}
		}
	})
	b.Run("no action", func(b *testing.B) {
		b.ReportAllocs()
		s := State{ZoneState: ZoneState{false, true}}
		for b.Loop() {
			a, err = r.Evaluate(s)
			if err != nil {
				b.Fatal(err)
			}
			if !a.IsState(State{ZoneState: ZoneState{false, true}}) {
				b.Fatal("unexpected result")
			}
		}
	})
}
