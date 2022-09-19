package stack_test

import (
	"bytes"
	"context"
	"github.com/clambin/tado-exporter/configuration"
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
	cfg, err := configuration.LoadConfiguration(bytes.NewBufferString(config))
	require.NoError(t, err)

	s, err := stack.New(cfg)
	require.NoError(t, err)

	mockPoller := &pollMock.Poller{}
	mockPoller.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	mockPoller.On("Run", mock.Anything, mock.Anything).Return(nil)
	mockPoller.On("Refresh").Return(nil)

	s.Poller = mockPoller

	mockBot := &slackMock.SlackBot{}
	mockBot.On("Run", mock.Anything).Return(nil)
	s.TadoBot = mockBot

	ctx, cancel := context.WithCancel(context.Background())

	s.Start(ctx)

	assert.Eventually(t, func() bool {
		resp, err2 := http.Get("http://localhost:8080/metrics")
		return err2 == nil && resp.StatusCode == http.StatusOK
	}, 500*time.Millisecond, 10*time.Millisecond)

	cancel()
	s.Stop()
}
