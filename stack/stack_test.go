package stack_test

import (
	"bytes"
	"context"
	"github.com/clambin/tado-exporter/configuration"
	slackbot "github.com/clambin/tado-exporter/controller/slackbot/mocks"
	poller "github.com/clambin/tado-exporter/poller/mocks"
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
	config := bytes.NewBufferString(`foo: bar`)

	cfg, err := configuration.LoadConfiguration(config)
	require.NoError(t, err)

	_, err = stack.New(cfg)
	assert.Error(t, err)
}

func TestStack(t *testing.T) {
	_ = os.Setenv("TADO_USERNAME", "test@example.com")
	_ = os.Setenv("TADO_PASSWORD", "some-password")
	config := `interval: 30s
exporter:
  enabled: true
  port: 9091
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
	cfg, err := configuration.LoadConfiguration(bytes.NewBufferString(config))
	require.NoError(t, err)

	s, err := stack.New(cfg)
	require.NoError(t, err)

	mockPoller := poller.NewPoller(t)
	//mockPoller.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	mockPoller.On("Run", mock.Anything, mock.Anything).Return(nil)
	//mockPoller.On("Refresh").Return(nil)

	s.Poller = mockPoller

	bot := slackbot.NewSlackBot(t)
	//bot.On("Register", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	bot.On("Run", mock.Anything).Return(nil)
	s.TadoBot = bot

	ctx, cancel := context.WithCancel(context.Background())

	s.Start(ctx)

	assert.Eventually(t, func() bool {
		_, err2 := http.Get("http://localhost:8080/")
		return err2 == nil
	}, time.Second, 10*time.Millisecond)

	assert.Eventually(t, func() bool {
		resp, err2 := http.Get("http://localhost:9091/metrics")
		return err2 == nil && resp.StatusCode == http.StatusOK
	}, time.Second, 10*time.Millisecond)

	//time.Sleep(time.Minute)
	cancel()
	s.Stop()
}
