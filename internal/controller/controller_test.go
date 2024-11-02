package controller_test

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestController_Run(t *testing.T) {
	t.Skip("flaky test")
	zoneCfg := configuration.Configuration{
		Home: configuration.HomeConfiguration{
			AutoAway: configuration.AutoAwayConfiguration{
				Users: []string{"A"},
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

	l := slog.New(slog.NewTextHandler(io.Discard, nil))

	m := controller.New(nil, zoneCfg, nil, p, l)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- m.Run(ctx) }()

	assert.Eventually(t, func() bool {
		return subscribers.Load() >= 2
	}, time.Second, time.Millisecond)

	ch <- poller.Update{
		HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
		HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
		Zones: poller.Zones{
			{
				Zone: tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
				ZoneState: tado.ZoneState{
					Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}},
					Overlay: &tado.ZoneOverlay{
						Termination: &oapi.TerminationManual,
					},
				},
			},
		},
		MobileDevices: []tado.MobileDevice{
			{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
		},
	}

	// TODO: flaky test
	assert.Eventually(t, func() bool {
		tasks := m.ReportTasks()
		return len(tasks) == 1
	}, 2*time.Second, 100*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}
