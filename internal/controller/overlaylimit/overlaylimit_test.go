package overlaylimit

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func makeTadoData() scheduler.TadoData {
	return scheduler.TadoData{
		Zone: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
			2: {ID: 2, Name: "bar"},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {},
			2: {Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING"},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}},
		},
	}
}

func TestOverlayLimit(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(`
controller:
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 1h
  - zoneName: "bar"
    maxTime: 1h
  - zoneName: "not-a-zone"
    maxTime: 1h
`))

	if assert.Nil(t, err) && assert.NotNil(t, cfg) && assert.NotNil(t, cfg.Controller.OverlayLimitRules) {

		schedule := scheduler.Scheduler{}
		setter := make(chan tadosetter.RoomCommand)
		limiter := OverlayLimit{
			Updates:    schedule.Register(),
			RoomSetter: setter,
			Slack:      make(tadobot.PostChannel),
			Rules:      *cfg.Controller.OverlayLimitRules,
		}
		go limiter.Run()

		log.SetLevel(log.DebugLevel)

		// set up the initial state
		err = schedule.Notify(makeTadoData())
		assert.Nil(t, err)

		slackMsgs := <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Manual temperature setting detected in zone bar", slackMsgs[0].Text)

		// fake the zone expiring its timer
		// not exactly thread-safe, but at this point overlayLimit is waiting on the next update
		details, ok := limiter.zoneDetails[2]
		if assert.True(t, ok) {
			details.expiryTimer = time.Now().Add(-1 * time.Hour)
			limiter.zoneDetails[2] = details
		}
		_ = schedule.Notify(makeTadoData())

		slackMsgs = <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Disabling manual temperature setting in zone bar", slackMsgs[0].Text)

		cmd := <-limiter.RoomSetter

		assert.Equal(t, 2, cmd.ZoneID)
		assert.True(t, cmd.Auto)
	}
}
