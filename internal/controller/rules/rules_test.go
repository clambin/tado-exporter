package rules

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/luart"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"testing"
	"time"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestRules_zoneRules(t *testing.T) {
	r, err := LoadZoneRules("zone", []RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "autoaway.lua"}, Users: []string{"user"}},
		{Name: "limitOverlay", Script: ScriptConfig{Packaged: "limitoverlay.lua"}},
		{Name: "nighttime.lua", Script: ScriptConfig{Packaged: "nighttime.lua"}, Args: map[string]any{"StartHour": 23, "StartMin": 0, "EndHour": 6, "EndMin": 0}},
	})
	require.NoError(t, err)
	require.Len(t, r.rules, 3)
	zr, ok := r.rules[2].(zoneRule)
	require.True(t, ok)
	luart.Register(zr.luaScript.State, func() time.Time {
		return time.Date(2024, time.December, 6, 13, 0, 0, 0, time.Local)
	})

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
