package zone_test

import (
	"context"
	"github.com/clambin/tado"
	mocks3 "github.com/clambin/tado-exporter/internal/controller/notifier/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/zone"
	"github.com/clambin/tado-exporter/internal/poller"
	mocks2 "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado/testutil"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestZoneController(t *testing.T) {
	api := mocks.NewTadoSetter(t)

	p := mocks2.NewPoller(t)
	pCh := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(pCh)
	p.EXPECT().Unsubscribe(pCh).Return()

	b := mocks3.NewSlackSender(t)

	cfg := configuration.ZoneConfiguration{
		Name: "room",
		Rules: configuration.ZoneRuleConfiguration{
			LimitOverlay: configuration.LimitOverlayConfiguration{
				Delay: time.Hour,
			}},
	}

	z := zone.New(api, p, b, cfg, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		errCh <- z.Run(ctx)
	}()

	testCases := []struct {
		update poller.Update
		event  []slack.Attachment
	}{
		{
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
			},
		},
		{
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay())},
			},
			event: []slack.Attachment{{Color: "good", Title: "room: moving to auto mode in 1h0m0s", Text: "manual temp setting detected"}},
		},
		{
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
			},
			event: []slack.Attachment{{Color: "good", Title: "room: canceling moving to auto mode", Text: "no manual temp setting detected"}},
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
