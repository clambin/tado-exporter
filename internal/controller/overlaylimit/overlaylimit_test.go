package overlaylimit_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/overlaylimit"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
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

const config = `
controller:
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 0s
  - zoneName: "bar"
    maxTime: 1h
  - zoneName: "not-a-zone"
    maxTime: 1h
`

func TestOverlayLimit(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(config))

	if assert.Nil(t, err) && assert.NotNil(t, cfg) && assert.NotNil(t, cfg.Controller.OverlayLimitRules) {

		schedule := scheduler.Scheduler{}
		setter := make(chan tadosetter.RoomCommand)
		limiter := overlaylimit.OverlayLimit{
			Updates:    schedule.Register(),
			RoomSetter: setter,
			Commands:   make(commands.RequestChannel, 5),
			Slack:      make(slackbot.PostChannel, 5),
			Rules:      *cfg.Controller.OverlayLimitRules,
		}
		go limiter.Run()

		// set up the initial state
		schedule.Update(makeTadoData())

		slackMsgs := <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Manual temperature setting detected in bar", slackMsgs[0].Title)

		// fake the zone expiring its timer
		// not exactly thread-safe, but at this point overlayLimit is waiting on the next update
		tadoData := makeTadoData()
		if zoneInfo, ok := tadoData.ZoneInfo[1]; assert.True(t, ok) {
			zoneInfo.Overlay = tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING"},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}
			tadoData.ZoneInfo[1] = zoneInfo
		}
		schedule.Update(tadoData)

		slackMsgs = <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Manual temperature setting detected in foo", slackMsgs[0].Title)

		schedule.Update(tadoData)

		// limiter writes to slack first and only then to roomsetter.
		// check buffering works by reading roomsetter & slack in the inverse order
		cmd := <-limiter.RoomSetter
		assert.Equal(t, 1, cmd.ZoneID)
		assert.True(t, cmd.Auto)

		slackMsgs = <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Setting foo back to auto mode", slackMsgs[0].Title)

		// test report command
		response := make(commands.ResponseChannel, 1)
		limiter.Commands <- commands.Command{
			Command:  commands.Report,
			Response: response,
		}
		output, gotResponse := <-response

		if assert.True(t, gotResponse) && assert.Len(t, output, 1) {
			sort.Strings(output)
			assert.Equal(t, "bar will be reset to auto in 1h0m0s", output[0])
		}

	}
}

func BenchmarkOverlayLimit_Run(b *testing.B) {
	cfg, err := configuration.LoadConfiguration([]byte(`
controller:
  overlayLimitRules:
    - zoneName: "foo"
      maxTime: 0s
`))

	if assert.Nil(b, err) && assert.NotNil(b, cfg) && assert.NotNil(b, cfg.Controller.OverlayLimitRules) {

		schedule := scheduler.Scheduler{}
		setter := make(chan tadosetter.RoomCommand)
		limiter := overlaylimit.OverlayLimit{
			Updates:    schedule.Register(),
			RoomSetter: setter,
			Commands:   make(commands.RequestChannel, 5),
			Slack:      make(slackbot.PostChannel, 5),
			Rules:      *cfg.Controller.OverlayLimitRules,
		}
		go limiter.Run()

		withoutOverlay := makeTadoData()
		withOverlay := makeTadoData()
		if zoneInfo, ok := withOverlay.ZoneInfo[1]; assert.True(b, ok) {
			zoneInfo.Overlay = tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING"},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}
			withOverlay.ZoneInfo[1] = zoneInfo
		}

		for i := 0; i < 1000; i++ {
			// set up the initial state
			schedule.Update(withoutOverlay)

			// put in overlay
			schedule.Update(withOverlay)
			_ = <-limiter.Slack

			// expire
			schedule.Update(withOverlay)

			// validate
			_ = <-limiter.Slack
			cmd := <-setter

			assert.True(b, cmd.Auto)
			assert.Len(b, limiter.Slack, 0)
		}

	}
}
