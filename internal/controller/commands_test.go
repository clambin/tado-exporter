package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/tadoproxy"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestController_doRooms(t *testing.T) {
	control := Controller{
		proxy: tadoproxy.Proxy{
			API: &mockapi.MockAPI{},
		},
	}

	err := control.proxy.Refresh()
	assert.Nil(t, err)

	output := control.doRooms()
	if assert.Len(t, output, 1) && assert.Len(t, output[0], 2) {
		assert.Equal(t, "bar: 19.9ºC (target: 25.0ºC MANUAL)", output[0][0])
		assert.Equal(t, "foo: 19.9ºC (target: 20.0ºC MANUAL)", output[0][1])
	}
}

func TestController_doUsers(t *testing.T) {
	control := Controller{
		proxy: tadoproxy.Proxy{
			API: &mockapi.MockAPI{},
		},
	}
	err := control.proxy.Refresh()
	assert.Nil(t, err)

	output := control.doUsers()
	if assert.Len(t, output, 1) && assert.Len(t, output[0], 2) {
		assert.Equal(t, "bar: away", output[0][0])
		assert.Equal(t, "foo: home", output[0][1])
	}
}

func TestController_doRules(t *testing.T) {
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
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 1h
  - zoneName: "bar"
    maxTime: 1h
`))
	if assert.Nil(t, err); err == nil {
		control := Controller{
			Configuration: &cfg.Controller,
			proxy: tadoproxy.Proxy{
				API: &mockapi.MockAPI{},
			},
		}
		err = control.Run()
		if assert.Nil(t, err); err == nil {

			output := control.doRules()
			if assert.Len(t, output, 2) {
				if assert.Len(t, output[0], 2) {
					assert.Equal(t, "bar is away. will set bar to manual in 1h0m0s", output[0][0])
					assert.Equal(t, "foo is home", output[0][1])
				}
				if assert.Len(t, output[1], 1) {
					assert.Equal(t, "bar will be reset to auto in 1h0m0s", output[1][0])
				}
			}
		}
	}
}
