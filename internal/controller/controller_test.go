package controller_test

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func BenchmarkController_Run(b *testing.B) {
	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	postChannel := make(slackbot.PostChannel)
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   1 * time.Millisecond,
		},
	}}
	mgr, _ := zonemanager.New(server, zoneConfig, postChannel)
	c, _ := controller.NewWith(server, pollr, mgr, nil, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)

	b.ResetTimer()

	for i := 0; i < 100; i++ {
		_ = server.SetZoneOverlay(ctx, 2, 15.5)
		_ = <-postChannel
		_ = <-postChannel
	}
}

func TestController_Run(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Millisecond,
		},
	}}

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(server, zoneConfig, postChannel)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := controller.NewWith(server, pollr, mgr, nil, 25*time.Millisecond)
	go c.Run(ctx)

	log.SetLevel(log.DebugLevel)

	err = server.SetZoneOverlay(ctx, 2, 15.5)
	assert.True(t, zoneInOverlay(ctx, server, 2))

	_ = <-postChannel
	_ = <-postChannel

	assert.False(t, zoneInOverlay(ctx, server, 2))

}

func TestController_RevertedOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   1 * time.Hour,
		},
	}}

	server := &mockapi.MockAPI{}
	pollr := poller.New(server)
	postChannel := make(slackbot.PostChannel)
	mgr, err := zonemanager.New(server, zoneConfig, postChannel)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := controller.NewWith(server, pollr, mgr, nil, 25*time.Millisecond)
	go c.Run(ctx)

	log.SetLevel(log.DebugLevel)

	err = server.SetZoneOverlay(ctx, 2, 15.5)
	assert.Nil(t, err)

	_ = <-postChannel

	assert.True(t, zoneInOverlay(ctx, server, 2))

	err = server.DeleteZoneOverlay(ctx, 2)
	assert.Nil(t, err)

	msg := <-postChannel

	if assert.Len(t, msg, 1) {
		assert.Equal(t, "resetting rule for bar", msg[0].Title)
	}

	mgr.ReportTasks()
	msg = <-postChannel
	if assert.Len(t, msg, 1) {
		assert.Equal(t, "no rules have been triggered", msg[0].Text)
	}
}

func zoneInOverlay(ctx context.Context, server tado.API, zoneID int) bool {
	info, err := server.GetZoneInfo(ctx, zoneID)
	return err == nil && info.Overlay.Type != ""
}
