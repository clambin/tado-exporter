package controller_test

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	slackbot "github.com/clambin/tado-exporter/internal/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	pollerMocks "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

var (
	zoneCfg []rules.ZoneConfig
)

func TestController_Run(t *testing.T) {
	a := mocks.NewTadoSetter(t)

	p := pollerMocks.NewPoller(t)
	ch := make(chan *poller.Update, 1)
	p.EXPECT().Register().Return(ch)
	p.EXPECT().Unregister(ch)
	p.EXPECT().Refresh()

	b := slackbot.NewSlackBot(t)
	b.EXPECT().Register(mock.AnythingOfType("string"), mock.AnythingOfType("slackbot.CommandFunc"))

	c := controller.New(a, zoneCfg, b, p)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- c.Run(ctx) }()

	response := c.Commands.DoRefresh(context.Background())
	assert.Len(t, response, 1)

	assert.Eventually(t, func() bool {
		response := c.Commands.ReportUsers(context.Background())
		return len(response) > 0
	}, time.Minute, 100*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}
