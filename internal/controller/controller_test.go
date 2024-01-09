package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestController_Run(t *testing.T) {
	zoneCfg := configuration.Configuration{
		Home: configuration.HomeConfiguration{
			AutoAway: configuration.AutoAwayConfiguration{
				Users: []string{"A", "B"},
				Delay: time.Minute,
			},
		},
		Zones: []configuration.ZoneConfiguration{{
			Name: "room",
			Rules: configuration.ZoneRuleConfiguration{
				LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour},
			},
		}},
	}

	p := mockPoller.NewPoller(t)
	ch := make(chan poller.Update, 1)
	var subscribers atomic.Int32
	p.EXPECT().Subscribe().RunAndReturn(func() chan poller.Update {
		subscribers.Add(1)
		return ch
	})
	p.EXPECT().Unsubscribe(ch)

	opts := slog.HandlerOptions{Level: slog.LevelDebug}
	l := slog.New(slog.NewTextHandler(os.Stderr, &opts))

	m := controller.New(nil, zoneCfg, nil, p, l)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- m.Run(ctx) }()

	assert.Eventually(t, func() bool {
		return subscribers.Load() >= 2
	}, time.Second, time.Millisecond)

	ch <- poller.Update{
		Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay(), testutil.ZoneInfoTemperature(18, 22))},
		UserInfo: map[int]tado.MobileDevice{
			100: testutil.MakeMobileDevice(100, "A", testutil.Home(false)),
			110: testutil.MakeMobileDevice(110, "B", testutil.Home(false)),
		},
		Home: true,
	}

	assert.Eventually(t, func() bool {
		tasks := m.ReportTasks()
		return len(tasks) == 1
	}, time.Second, 100*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}
