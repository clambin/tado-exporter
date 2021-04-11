package controller_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func BenchmarkController_Run(b *testing.B) {
	server := &mockapi.MockAPI{}
	pollr := poller.New(server, 10*time.Millisecond)
	postChannel := make(slackbot.PostChannel)
	schedulr := scheduler.New(server, postChannel)
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Millisecond,
		},
	}}
	mgr, _ := zonemanager.New(server, zoneConfig, pollr.Update, schedulr)
	c, _ := controller.NewWith(server, pollr, mgr, schedulr, nil)
	go c.Run()

	b.ResetTimer()

	for i := 0; i < 10; i++ {
		_ = server.SetZoneOverlay(2, 15.5)
		_ = <-postChannel
		_ = <-postChannel
	}
}

func TestController_Run(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   50 * time.Millisecond,
		},
	}}

	server := &mockapi.MockAPI{}
	pollr := poller.New(server, 25*time.Millisecond)
	postChannel := make(slackbot.PostChannel, 200)
	schedulr := scheduler.New(server, postChannel)
	mgr, err := zonemanager.New(server, zoneConfig, pollr.Update, schedulr)

	if assert.Nil(t, err) {
		c, _ := controller.NewWith(server, pollr, mgr, schedulr, nil)

		log.SetLevel(log.DebugLevel)

		err = server.SetZoneOverlay(2, 15.5)
		assert.Nil(t, err)
		var info tado.ZoneInfo
		info, err = server.GetZoneInfo(2)
		assert.Nil(t, err)
		assert.Equal(t, "MANUAL", info.Overlay.Type)

		go c.Run()

		assert.Eventually(t, func() bool {
			info, err = server.GetZoneInfo(2)
			return err == nil && info.Overlay.Type == ""
		}, 100*time.Millisecond, 10*time.Millisecond)

		assert.Len(t, postChannel, 2)

		c.Stop()
	}
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
	pollr := poller.New(server, 25*time.Millisecond)
	postChannel := make(slackbot.PostChannel, 200)
	schedulr := scheduler.New(server, postChannel)
	mgr, err := zonemanager.New(server, zoneConfig, pollr.Update, schedulr)

	if assert.Nil(t, err) {
		c, _ := controller.NewWith(server, pollr, mgr, schedulr, nil)
		go c.Run()

		log.SetLevel(log.DebugLevel)

		err = server.SetZoneOverlay(2, 15.5)
		assert.Nil(t, err)

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == models.ZoneAuto
		}, 500*time.Millisecond, 10*time.Millisecond)

		err = server.DeleteZoneOverlay(2)
		assert.Nil(t, err)

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == models.ZoneUnknown
		}, 500*time.Hour, 10*time.Millisecond)

		c.Stop()
	}
}
