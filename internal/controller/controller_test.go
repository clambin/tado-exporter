package controller_test

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestController_Run(t *testing.T) {
	zoneCfg := configuration.Configuration{
		Zones: []configuration.ZoneConfiguration{{
			Name: "foo",
			Rules: configuration.ZoneRuleConfiguration{
				LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour},
			},
		}},
	}

	a := mocks.NewTadoSetter(t)
	p := mockPoller.NewPoller(t)
	ch := make(chan poller.Update, 1)
	p.EXPECT().Subscribe().Return(ch)
	p.EXPECT().Unsubscribe(ch)

	m := controller.New(a, zoneCfg, nil, p, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- m.Run(ctx) }()

	cancel()
	assert.NoError(t, <-errCh)
}
