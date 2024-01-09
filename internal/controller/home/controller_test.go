package home_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/home"
	mocks3 "github.com/clambin/tado-exporter/internal/controller/notifier/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	mocks2 "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestHomeController(t *testing.T) {
	api := mocks.NewTadoSetter(t)

	p := mocks2.NewPoller(t)
	pCh := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(pCh)
	p.EXPECT().Unsubscribe(pCh).Return()

	b := mocks3.NewSlackSender(t)

	cfg := configuration.HomeConfiguration{AutoAway: configuration.AutoAwayConfiguration{
		Users: []string{"A"},
		Delay: time.Hour,
	}}

	h := home.New(api, p, b, cfg, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		errCh <- h.Run(ctx)
	}()

	testCases := []struct {
		update poller.Update
		event  []slack.Attachment
	}{
		{
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(10, "A", testutil.Home(true)),
				},
				Home: true,
			},
		},
		{
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(10, "A", testutil.Home(false)),
				},
				Home: true,
			},
			event: []slack.Attachment{{Color: "good", Title: "setting home to away mode in 1h0m0s", Text: "A is away"}},
		},
		{
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(10, "A", testutil.Home(true)),
				},
				Home: true,
			},
			event: []slack.Attachment{{Color: "good", Title: "canceling setting home to away mode", Text: "A is home"}},
		},
	}

	for _, tt := range testCases {
		var done chan struct{}

		if tt.event != nil {
			done = make(chan struct{})
			b.EXPECT().Send("", tt.event).RunAndReturn(func(_ string, attachments []slack.Attachment) error {
				done <- struct{}{}
				return nil
			}).Once()
		}
		pCh <- tt.update
		if tt.event != nil {
			<-done
		}
	}

	cancel()
	assert.NoError(t, <-errCh)
}
