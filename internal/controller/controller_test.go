package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado-exporter/pkg/pubsub"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
)

func TestNew(t *testing.T) {
	cfg := Configuration{
		Home: []RuleConfiguration{
			{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"foo"}},
		},
		Zones: map[string][]RuleConfiguration{
			"living room": {
				{Name: "autoAway", Script: ScriptConfig{Packaged: "autoaway.lua"}, Users: []string{"foo"}},
				{Name: "limitOverlay", Script: ScriptConfig{Packaged: "limitoverlay.lua"}, Users: []string{"foo"}},
			},
		},
	}

	m, err := New(cfg, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Len(t, m.controllers, 2)
}

func TestController_Run(t *testing.T) {
	cfg := Configuration{
		Home: []RuleConfiguration{
			{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"user A"}},
		},
		Zones: map[string][]RuleConfiguration{
			"my room": {
				{Name: "autoAway", Script: ScriptConfig{Packaged: "autoaway.lua"}, Users: []string{"user A"}},
				{Name: "limitOverlay", Script: ScriptConfig{Packaged: "limitoverlay.lua"}},
			},
		},
	}
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	p := pubsub.New[poller.Update](l)
	n := fakeNotifier{ch: make(chan string)}

	m, err := New(cfg, p, nil, &n, l)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Len(t, m.controllers, 2)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- m.Run(ctx) }()

	require.Eventually(t, func() bool {
		return p.Subscribers() > 0
	}, time.Second, time.Millisecond)

	u := testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME),
		testutils.WithZone(10, "my room", tado.PowerON, 18, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
		testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
	)
	go p.Publish(u)

	const want = "*my room*: switching heating to auto mode in 1h0m0s\nReason: manual setting detected"
	msg := <-n.ch
	assert.Equal(t, want, msg)
	assert.Equal(t, want, strings.Join(m.ReportTasks(), ", "))

	cancel()
	assert.NoError(t, <-errCh)
}
