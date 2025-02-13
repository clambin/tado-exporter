package controller

import (
	"github.com/clambin/go-common/pubsub"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

var (
	discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
)

func TestNew(t *testing.T) {
	cfg := Configuration{
		Home: []rules.RuleConfiguration{
			{Name: "autoAway", Script: rules.ScriptConfig{Packaged: "homeandaway"}, Users: []string{"foo"}},
		},
		Zones: map[string][]rules.RuleConfiguration{
			"living room": {
				{Name: "autoAway", Script: rules.ScriptConfig{Packaged: "autoaway"}, Users: []string{"foo"}},
				{Name: "limitOverlay", Script: rules.ScriptConfig{Packaged: "limitoverlay"}, Users: []string{"foo"}},
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
		Home: []rules.RuleConfiguration{
			{Name: "autoAway", Script: rules.ScriptConfig{Packaged: "homeandaway"}, Users: []string{"user A"}},
		},
		Zones: map[string][]rules.RuleConfiguration{
			"my room": {
				{Name: "autoAway", Script: rules.ScriptConfig{Packaged: "autoaway"}, Users: []string{"user A"}},
				{Name: "limitOverlay", Script: rules.ScriptConfig{Packaged: "limitoverlay"}},
				{Name: "limitOverlay", Script: rules.ScriptConfig{Packaged: "limitoverlay"}},
			},
		},
	}
	//l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	l := slog.New(slog.DiscardHandler)
	var p pubsub.Publisher[poller.Update]
	n := fakeNotifier{ch: make(chan string)}

	m, err := New(cfg, &p, nil, &n, l)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Len(t, m.controllers, 2)

	go func() { assert.NoError(t, m.Run(t.Context())) }()

	require.Eventually(t, func() bool {
		return p.Subscribers() > 0
	}, time.Second, time.Millisecond)

	u := testutils.Update(
		testutils.WithZone(10, "my room", tado.PowerON, 18, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
		testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
	)
	go p.Publish(u)

	const want = "*my room*: switching heating to auto mode in 1h0m0s\nReason: manual setting detected"
	msg := <-n.ch
	assert.Equal(t, want, msg)
	assert.Equal(t, want, strings.Join(m.ReportTasks(), ", "))
}
