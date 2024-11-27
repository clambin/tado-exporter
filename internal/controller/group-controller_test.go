package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestGroupController(t *testing.T) {
	ruleConfig := []RuleConfiguration{
		{
			Name:   "autoAway",
			Script: ScriptConfig{Packaged: `autoaway.lua`},
			Users:  []string{"user A"},
		},
		{
			Name: "limitOverlay",
			Script: ScriptConfig{Text: `function Evaluate(_, zone, _)
	if zone == "auto" then
		return "auto", 0, "no manual setting detected"
	end
	return "auto", 0, "manual setting detected"
end
`},
		},
	}
	zoneRules, err := loadZoneRules("zone", ruleConfig)
	require.NoError(t, err)
	require.Len(t, zoneRules.rules, 2)

	ctx := context.Background()
	//f := fakeNotifier{ch: make(chan string)}
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tadoClient := mocks.NewTadoClient(t)
	g := newGroupController(zoneRules, nil, tadoClient, nil, l)

	u := testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME),
		testutils.WithZone(1, "zone", tado.PowerON, 21, 18),
		testutils.WithMobileDevice(1, "user A", testutils.WithLocation(false, true)),
	)

	a, ok := g.processUpdate(u)
	assert.True(t, ok)
	assert.Equal(t, 15*time.Minute, a.GetDelay())

	g.scheduleJob(ctx, a)
	j := g.scheduledJob.Load()
	require.NotNil(t, j)
	assert.Equal(t, 15*time.Minute, j.GetDelay().Round(time.Minute))

	u = testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME),
		testutils.WithZone(1, "zone", tado.PowerON, 21, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
		testutils.WithMobileDevice(1, "user A", testutils.WithLocation(false, true)),
	)

	a, ok = g.processUpdate(u)
	assert.True(t, ok)
	assert.Equal(t, time.Duration(0), a.GetDelay())

	tadoClient.EXPECT().
		DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(1)).
		Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
		Once()
	g.scheduleJob(ctx, a)
}
