package controller

import (
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
)

func TestGroupController(t *testing.T) {
	ruleConfig := []RuleConfiguration{
		{
			Name:   "autoAway",
			Script: ScriptConfig{Packaged: "autoaway.lua"},
			Users:  []string{"user A"},
		},
		{
			Name:   "limitOverlay",
			Script: ScriptConfig{Packaged: "limitoverlay.lua"},
		},
	}
	zoneRules, err := LoadZoneRules("zone", ruleConfig)
	require.NoError(t, err)
	require.Len(t, zoneRules.rules, 2)

	f := fakeNotifier{ch: make(chan string)}
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	g := newGroupController(zoneRules, nil, nil, &f, l)

	u := testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME),
		testutils.WithZone(1, "zone", tado.PowerON, 21, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
		testutils.WithMobileDevice(1, "user A", testutils.WithLocation(false, true)),
	)

	a, ok := g.processUpdate(u)
	assert.True(t, ok)
	t.Log(a.Description(true), a.GetReason())
}
