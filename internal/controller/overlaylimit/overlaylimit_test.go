package overlaylimit_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/overlaylimit"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/internal/tadobot"
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

func TestOverlayLimit(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(`
controller:
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 0s
  - zoneName: "bar"
    maxTime: 1h
  - zoneName: "not-a-zone"
    maxTime: 1h
`))

	if assert.Nil(t, err) && assert.NotNil(t, cfg) && assert.NotNil(t, cfg.Controller.OverlayLimitRules) {

		schedule := scheduler.Scheduler{}
		setter := make(chan tadosetter.RoomCommand)
		limiter := overlaylimit.OverlayLimit{
			Updates:    schedule.Register(),
			RoomSetter: setter,
			Commands:   make(commands.RequestChannel, 5),
			Slack:      make(tadobot.PostChannel, 5),
			Rules:      *cfg.Controller.OverlayLimitRules,
		}
		go limiter.Run()

		// set up the initial state
		schedule.Notify(makeTadoData())

		slackMsgs := <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Manual temperature setting detected in zone bar", slackMsgs[0].Text)

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
		schedule.Notify(tadoData)

		slackMsgs = <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Manual temperature setting detected in zone foo", slackMsgs[0].Text)

		schedule.Notify(tadoData)

		// limiter writes to slack first and only then to roomsetter.
		// check buffering works by reading roomsetter & slack in the inverse order
		cmd := <-limiter.RoomSetter
		assert.Equal(t, 1, cmd.ZoneID)
		assert.True(t, cmd.Auto)

		slackMsgs = <-limiter.Slack
		assert.Len(t, slackMsgs, 1)
		assert.Equal(t, "Disabling manual temperature setting in zone foo", slackMsgs[0].Text)

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
