package controller_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/scheduler/mockscheduler"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

/*
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
	mgr, _ := zonemanager.New(server, zoneConfig, pollr.Update, schedulr.Schedule)
	c, _ := controller.NewWith(server, pollr, mgr, schedulr, nil)
	go c.Run()

	b.ResetTimer()

	for i := 0; i < 10; i++ {
		_ = server.SetZoneOverlay(2, 15.5)
		_ = <-postChannel
		_ = <-postChannel
		// wait for zone mgr to clear the queued state
		time.Sleep(15 * time.Millisecond)
	}
}
*/
func TestController_Run(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   100 * time.Millisecond,
		},
	}}

	server := &mockapi.MockAPI{}
	pollr := poller.New(server, 25*time.Millisecond)
	schedulr := mockscheduler.New()
	mgr, err := zonemanager.New(server, zoneConfig, pollr.Update, schedulr)

	if assert.Nil(t, err) {
		c, _ := controller.NewWith(server, pollr, mgr, schedulr, nil)

		log.SetLevel(log.DebugLevel)

		go c.Run()

		err = server.SetZoneOverlay(2, 15.5)
		assert.Nil(t, err)
		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Auto
		}, 50*time.Millisecond, 10*time.Millisecond)

		assert.Eventually(t, func() bool {
			return schedulr.ScheduledState(2).State == model.Unknown
		}, 500*time.Millisecond, 10*time.Millisecond)

		c.Stop()
	}
}
