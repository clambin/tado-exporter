package autoaway

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/actions"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayConfig(t *testing.T) {
	tadoData := scheduler.TadoData{
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
					AtHome: true,
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

		server := mockapi.MockAPI{}
		schedule := scheduler.Scheduler{}
		away := AutoAway{
			Actions: actions.Actions{
				API: &server,
			},
			Updates: schedule.Register(),
			Rules:   *cfg.Controller.AutoAwayRules,
		}

		err = away.process(&tadoData)
		assert.Nil(t, err)

		if assert.NotNil(t, away.deviceInfo) {

			assert.Len(t, away.deviceInfo, 2)

			var deviceInfo DeviceInfo

			// "bar" was previously home
			deviceInfo, _ = away.deviceInfo[2]
			oldActivationTime := deviceInfo.activationTime
			deviceInfo.state = autoAwayStateHome
			away.deviceInfo[2] = deviceInfo

			err = away.process(&tadoData)

			assert.Nil(t, err)
			assert.Equal(t, autoAwayState(autoAwayStateAway), away.deviceInfo[2].state)
			assert.True(t, away.deviceInfo[2].activationTime.After(oldActivationTime))

			// "bar" has been away for a long time
			deviceInfo, _ = away.deviceInfo[2]
			deviceInfo.activationTime = time.Now().Add(-2 * time.Hour)
			away.deviceInfo[2] = deviceInfo

			err = away.process(&tadoData)

			assert.Nil(t, err)
			assert.Equal(t, autoAwayState(autoAwayStateReported), away.deviceInfo[2].state)
			assert.Len(t, server.Overlays, 1)
			assert.Equal(t, 15.0, server.Overlays[2])

			// "foo" was previously away
			server.Overlays[1] = 15
			autoAway, _ := away.deviceInfo[1]
			autoAway.state = autoAwayStateAway
			away.deviceInfo[1] = autoAway

			err = away.process(&tadoData)
			assert.Nil(t, err)
			_, ok := server.Overlays[1]
			assert.False(t, ok)
		}
	}
}
