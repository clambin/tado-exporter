package controller

import (
	"context"
	"github.com/clambin/tado-exporter/controller/mocks"
	slackbot "github.com/clambin/tado-exporter/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/poller"
	mocks2 "github.com/clambin/tado-exporter/poller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

var (
	zoneCfg []rules.ZoneConfig
)

func TestController_Run(t *testing.T) {
	a := mocks.NewTadoSetter(t)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	ch := make(chan *poller.Update, 1)
	p := mocks2.NewPoller(t)
	p.On("Refresh").Return(nil)
	p.On("Register").Return(ch)
	p.On("Unregister", ch)

	b := slackbot.NewSlackBot(t)
	b.On("Register", mock.AnythingOfType("string"), mock.AnythingOfType("slackbot.CommandFunc")).Return(nil)

	c := New(a, zoneCfg, b, p)

	wg.Add(1)
	go func() { defer wg.Done(); _ = c.Run(ctx) }()

	response := c.cmds.DoRefresh(context.Background())
	assert.Len(t, response, 1)

	assert.Eventually(t, func() bool {
		response = c.cmds.ReportUsers(context.Background())
		return len(response) > 0
	}, time.Minute, 100*time.Millisecond)

	cancel()
	wg.Wait()
}
