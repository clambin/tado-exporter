package autoaway_test

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/autoaway"
	"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"sort"
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
		MobileDevice: map[int]tado.MobileDevice{
			1: {
				ID:       1,
				Name:     "foo",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{
					AtHome: true,
				},
			},
			2: {
				ID:       2,
				Name:     "bar",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			},
		},
	}
}

func TestAutoAwayRun(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(`
controller:
  autoAwayRules:
  - zoneName: "foo"
    mobileDeviceName: "foo"
    waitTime: 0s
    targetTemperature: 5.0
  - zoneName: "bar"
    mobileDeviceName: "bar"
    waitTime: 1h
    targetTemperature: 15.0
`))

	if assert.Nil(t, err) && assert.NotNil(t, cfg) && assert.NotNil(t, cfg.Controller.AutoAwayRules) {

		schedule := scheduler.Scheduler{}
		setter := make(chan tadosetter.RoomCommand, 4096)
		away := autoaway.AutoAway{
			RoomSetter: setter,
			Updates:    schedule.Register(),
			Commands:   make(commands.RequestChannel, 5),
			Slack:      make(slackbot.PostChannel, 5),
			Rules:      *cfg.Controller.AutoAwayRules,
		}
		go away.Run()

		// set up the initial state
		schedule.Update(makeTadoData())

		// device 2 was away. Now mark it being home
		tadoData := makeTadoData()
		device := tadoData.MobileDevice[2]
		device.Location.AtHome = true
		tadoData.MobileDevice[2] = device
		schedule.Update(tadoData)

		// resulting command should to set zone 2 to Auto
		msg := <-setter
		assert.True(t, msg.Auto)
		assert.Equal(t, 2, msg.ZoneID)

		// check slack output
		slackOutput := <-away.Slack
		if assert.Len(t, slackOutput, 1) {
			assert.Equal(t, "resetting bar to auto", slackOutput[0].Text)
		}

		// mark device 2 as away again
		schedule.Update(makeTadoData())

		// should not result in an action
		if assert.Eventually(t, func() bool { return len(setter) == 0 }, 500*time.Millisecond, 10*time.Millisecond) == false {
			panic("unexpected message expected in channel. aborting ...")
		}
		// check slack output
		slackOutput = <-away.Slack
		if assert.Len(t, slackOutput, 1) {
			assert.Equal(t, "will set bar to manual in 1h0m0s", slackOutput[0].Text)
		}

		// device 1 was home. mark it as away
		tadoData = makeTadoData()
		mobileDevice, ok := tadoData.MobileDevice[1]
		if assert.True(t, ok) {
			mobileDevice.Location.AtHome = false
			tadoData.MobileDevice[1] = mobileDevice
		}

		// run 2 status updates. the first sets the user as away.  the second will expire the timer
		schedule.Update(tadoData)
		slackOutput = <-away.Slack
		if assert.Len(t, slackOutput, 1) {
			assert.Equal(t, "will set foo to manual in 0s", slackOutput[0].Text)
		}
		schedule.Update(tadoData)

		// resulting command should be to set zone 1 to manual
		msg = <-setter
		assert.False(t, msg.Auto)
		assert.Equal(t, 1, msg.ZoneID)
		assert.Equal(t, 5.0, msg.Temperature)

		// check slack output
		slackOutput = <-away.Slack
		if assert.Len(t, slackOutput, 1) {
			assert.Equal(t, "activating manual control in zone foo", slackOutput[0].Text)
		}

		// test report command
		response := make(commands.ResponseChannel, 1)
		away.Commands <- commands.Command{
			Command:  commands.Report,
			Response: response,
		}
		output, gotResponse := <-response

		if assert.True(t, gotResponse) && assert.Len(t, output, 2) {
			sort.Strings(output)
			assert.Equal(t, "bar is away. will set bar to manual in 1h0m0s", output[0])
			assert.Equal(t, "foo is away. foo is set to manual", output[1])
		}
	}
}

func TestAutoAway_InvalidEntries(t *testing.T) {
	cfg, err := configuration.LoadConfiguration([]byte(`
controller:
  autoAwayRules:
  - zoneName: "foo"
    mobileDeviceName: "foo"
    waitTime: 0s
    targetTemperature: 5.0
  - zoneName: "bar"
    mobileDeviceName: "bad_bar"
    waitTime: 1h
    targetTemperature: 15.0
  - zoneName: "bad_bar"
    mobileDeviceName: "bar"
    waitTime: 1h
    targetTemperature: 15.0
`))

	if assert.Nil(t, err) && assert.NotNil(t, cfg) && assert.NotNil(t, cfg.Controller.AutoAwayRules) {

		schedule := scheduler.Scheduler{}
		setter := make(chan tadosetter.RoomCommand, 4096)
		away := autoaway.AutoAway{
			RoomSetter: setter,
			Updates:    schedule.Register(),
			Commands:   make(commands.RequestChannel, 5),
			Slack:      make(slackbot.PostChannel, 5),
			Rules:      *cfg.Controller.AutoAwayRules,
		}
		go away.Run()

		// set up the initial state
		tadoData := makeTadoData()
		device, _ := tadoData.MobileDevice[1]
		device.Location.Stale = true
		tadoData.MobileDevice[1] = device
		schedule.Update(tadoData)

		assert.Eventually(t, func() bool {
			req := commands.Command{
				Command:  commands.Report,
				Response: make(commands.ResponseChannel),
			}
			away.Commands <- req
			output := <-req.Response

			return len(output) == 1 && output[0] == "foo is undetermined"

		}, 1*time.Second, 10*time.Millisecond)
	}
}
