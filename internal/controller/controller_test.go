package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestController_Run(t *testing.T) {
	var (
		err  error
		cfg  *configuration.Configuration
		ctrl *Controller
	)
	cfg, err = configuration.LoadConfiguration([]byte(`
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
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 1h
  - zoneName: "bar"
    maxTime: 1h
`))
	if assert.Nil(t, err) && assert.NotNil(t, cfg) {
		ctrl, err = New("", "", "", &cfg.Controller)
		if assert.Nil(t, err) && assert.NotNil(t, ctrl) {

			ctrl.registry.API = &mockapi.MockAPI{}

			err = ctrl.Run()
			assert.Nil(t, err)

			// assert.Len(t, ctrl.AutoAwayInfo, 2)
			// assert.Len(t, ctrl.Overlays, 1)
		}
	}
}
