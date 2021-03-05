package autoaway

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/pkg/tado"
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
		MobileDevice: map[int]tado.MobileDevice{
			1: {
				ID:       1,
				Name:     "foo",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{
					Stale:  false,
					AtHome: false,
				},
			},
			2: {
				ID:       2,
				Name:     "bar",
				Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
				Location: tado.MobileDeviceLocation{
					Stale:  false,
					AtHome: false,
				},
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
    waitTime: 1h
    targetTemperature: 5.0
  - zoneName: "bar"
    mobileDeviceName: "bar"
    waitTime: 1h
    targetTemperature: 15.0
`))

	if assert.Nil(t, err) && assert.NotNil(t, cfg) && assert.NotNil(t, cfg.Controller.AutoAwayRules) {

		schedule := scheduler.Scheduler{}
		setter := make(chan tadosetter.RoomCommand, 4096)
		away := AutoAway{
			RoomSetter: setter,
			Updates:    schedule.Register(),
			Rules:      *cfg.Controller.AutoAwayRules,
		}
		go away.Run()

		// set up the initial state
		err = schedule.Notify(makeTadoData())
		assert.Nil(t, err)

		// device 2 was away. Now mark it being home
		tadoData := makeTadoData()
		device := tadoData.MobileDevice[2]
		device.Location.AtHome = true
		tadoData.MobileDevice[2] = device
		err = schedule.Notify(tadoData)
		assert.Nil(t, err)

		// resulting command should to set zone 2 to Auto
		msg := <-setter
		assert.True(t, msg.Auto)
		assert.Equal(t, 2, msg.ZoneID)

		// mark device 2 as away again
		err = schedule.Notify(makeTadoData())
		assert.Nil(t, err)

		// should not result in an action
		if assert.Eventually(t, func() bool { return len(setter) == 0 }, 500*time.Millisecond, 10*time.Millisecond) == false {
			panic("unexpected message expected in channel. aborting ...")
		}

		// fake the device being away for a long time
		// not exactly thread-safe, but at this point autoAway is waiting on the next update
		deviceInfo, ok := away.deviceInfo[2]
		if assert.True(t, ok) {
			deviceInfo.activationTime = time.Now().Add(-12 * time.Hour)
			away.deviceInfo[2] = deviceInfo
		}

		// run another status
		err = schedule.Notify(makeTadoData())
		assert.Nil(t, err)

		// resulting command should be to set zone2 to overlay
		msg = <-setter
		assert.False(t, msg.Auto)
		assert.Equal(t, 2, msg.ZoneID)
		assert.Equal(t, 15.0, msg.Temperature)
	}
}
