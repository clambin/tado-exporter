package stack_test

import (
	"bou.ke/monkey"
	"context"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/setter"
	pollMock "github.com/clambin/tado-exporter/poller/mocks"
	slackMock "github.com/clambin/tado-exporter/slackbot/mocks"
	"github.com/clambin/tado-exporter/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestNew_Errors(t *testing.T) {
	config := ``

	cfg, err := configuration.LoadConfiguration([]byte(config))
	require.NoError(t, err)

	fakeExit := func(int) {
		panic("os.Exit called")
	}
	patch := monkey.Patch(os.Exit, fakeExit)
	defer patch.Unpatch()

	require.Panics(t, func() { _ = stack.New(cfg) })
}

func TestStack(t *testing.T) {
	_ = os.Setenv("TADO_USERNAME", "test@example.com")
	_ = os.Setenv("TADO_PASSWORD", "some-password")
	config := `interval: 30s
exporter:
  enabled: true
  port: 8080
controller:
  enabled: true
  tadoBot:
    enabled: false
    token: "some-token"
  zones:
    - name: "Study"
      limitOverlay:
        enabled: true
        delay: 1h
`
	cfg, err := configuration.LoadConfiguration([]byte(config))
	require.NoError(t, err)

	s := stack.New(cfg)
	require.NotNil(t, s)

	mockPoller := &pollMock.Poller{}
	mockPoller.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	mockPoller.On("Run", mock.Anything, mock.Anything).Return(nil)
	s.Poller = mockPoller

	mockBot := &slackMock.SlackBot{}
	mockBot.On("Run", mock.Anything).Return(nil)
	s.TadoBot = mockBot
	s.Controller.Setter.(*setter.Server).SlackBot = mockBot

	ctx, cancel := context.WithCancel(context.Background())

	s.Start(ctx)

	assert.Eventually(t, func() bool {
		resp, err2 := http.Get("http://localhost:8080/metrics")
		return err2 == nil && resp.StatusCode == http.StatusOK
	}, 500*time.Millisecond, 10*time.Millisecond)

	cancel()
	s.Stop()
}
