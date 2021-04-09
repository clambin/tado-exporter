package controller_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/zonemanager"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestController_Run(t *testing.T) {
	server := &mockapi.MockAPI{}
	pollr := poller.New(server, 25*time.Millisecond)

	postChannel := make(slackbot.PostChannel, 20)
	schedulr := scheduler.New(server, postChannel)

	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Millisecond,
		},
	}}

	mgr, err := zonemanager.New(server, zoneConfig, pollr.Update, schedulr.Register)

	if assert.Nil(t, err) {
		c, _ := controller.NewWith(server, pollr, mgr, schedulr, nil)

		log.SetLevel(log.DebugLevel)

		go c.Run()

		err = server.SetZoneOverlay(2, 15.5)
		assert.Nil(t, err)
		assert.Eventually(t, func() bool {
			if zoneInfo, err := server.GetZoneInfo(2); err == nil {
				return zoneInfo.Overlay.Setting.Temperature.Celsius == 15.5
			}
			return false
		}, 500*time.Millisecond, 10*time.Millisecond)

		time.Sleep(100 * time.Millisecond)

		assert.Eventually(t, func() bool {
			if zoneInfo, err := server.GetZoneInfo(2); err == nil {
				return zoneInfo.Overlay.Type == ""
			}
			return false
		}, 500*time.Millisecond, 10*time.Millisecond)

		assert.Len(t, postChannel, 1)

		c.Stop()
	}
}