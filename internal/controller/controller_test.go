package controller_test

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	pollerMocks "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

var zoneCfg = []rules.ZoneConfig{
	{
		Zone: "foo",
		Rules: []rules.RuleConfig{
			{
				Kind:  rules.LimitOverlay,
				Delay: time.Hour,
			},
		},
	},
}

func TestController_Run(t *testing.T) {
	a := mocks.NewTadoSetter(t)

	p := pollerMocks.NewPoller(t)
	ch := make(chan *poller.Update, 1)
	p.EXPECT().Subscribe().Return(ch)
	p.EXPECT().Unsubscribe(ch)

	c := controller.New(a, zoneCfg, nil, p, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- c.Run(ctx) }()

	cancel()
	assert.NoError(t, <-errCh)
}
