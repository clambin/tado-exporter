package autoaway

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/registry"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TODO: adapt as per overlimit_test.go to avoid timing issues

func TestAutoAwayConfig(t *testing.T) {
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
		reg := registry.Registry{
			API: &server,
		}

		away := AutoAway{
			Updates:  reg.Register(),
			Registry: &reg,
			Rules:    *cfg.Controller.AutoAwayRules,
		}
		go func() { away.Run() }()

		err = reg.Run()
		assert.Nil(t, err)

		assert.Eventually(t, func() bool { return away.deviceInfo != nil }, 500*time.Millisecond, 50*time.Millisecond)
		assert.Len(t, away.deviceInfo, 2)

		var deviceInfo DeviceInfo

		// "bar" was previously home
		deviceInfo, _ = away.deviceInfo[2]
		oldActivationTime := deviceInfo.activationTime
		deviceInfo.state = autoAwayStateHome
		away.deviceInfo[2] = deviceInfo

		err = reg.Run()
		assert.Nil(t, err)

		assert.Eventually(t, func() bool {
			deviceInfo, _ = away.deviceInfo[2]
			return deviceInfo.state == autoAwayState(autoAwayStateAway) && deviceInfo.activationTime.After(oldActivationTime)
		}, 500*time.Millisecond, 100*time.Millisecond)

		// "bar" has been away for a long time
		deviceInfo, _ = away.deviceInfo[2]
		deviceInfo.activationTime = time.Now().Add(-2 * time.Hour)
		away.deviceInfo[2] = deviceInfo

		err = reg.Run()
		assert.Nil(t, err)

		assert.Eventually(t, func() bool {
			deviceInfo, _ = away.deviceInfo[2]
			return deviceInfo.state == autoAwayState(autoAwayStateReported) &&
				len(server.Overlays) == 1 &&
				server.Overlays[2] == 15.0
		}, 500*time.Millisecond, 100*time.Millisecond)

		// TODO: once bar is away, it shouldn't trigger any more events

		// "foo" was previously away
		server.Overlays[1] = 15
		autoAway, _ := away.deviceInfo[1]
		autoAway.state = autoAwayStateAway
		away.deviceInfo[1] = autoAway

		err = reg.Run()
		assert.Nil(t, err)
		assert.Eventually(t, func() bool { _, ok := server.Overlays[1]; return !ok }, 500*time.Millisecond, 100*time.Millisecond)
	}
}
