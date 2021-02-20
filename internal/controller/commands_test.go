package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/tadoproxy"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"strings"
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
	if assert.Len(t, output, 1) {
		lines := strings.Split(output[0].Text, "\n")
		if assert.Len(t, lines, 2) {
			assert.Equal(t, "bar: 19.9ºC (target: 25.0ºC MANUAL)", lines[0])
			assert.Equal(t, "foo: 19.9ºC (target: 20.0ºC MANUAL)", lines[1])
		}
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
	if assert.Len(t, output, 1) {
		lines := strings.Split(output[0].Text, "\n")
		if assert.Len(t, lines, 2) {
			assert.Equal(t, "bar: away", lines[0])
			assert.Equal(t, "foo: home", lines[1])
		}
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
				lines := strings.Split(output[0].Text, "\n")
				if assert.Len(t, lines, 2) {
					assert.Equal(t, "bar is away. will set bar to manual in 1h0m0s", lines[0])
					assert.Equal(t, "foo is home", lines[1])
				}

				lines = strings.Split(output[1].Text, "\n")
				if assert.Len(t, lines, 1) {
					assert.Equal(t, "bar will be reset to auto in 1h0m0s", lines[0])
				}
			}
		}
	}
}

func TestController_doSetTemperature(t *testing.T) {
	control := Controller{
		Configuration: &configuration.ControllerConfiguration{},
		proxy: tadoproxy.Proxy{
			API: &mockapi.MockAPI{},
		},
	}

	err := control.proxy.Refresh()
	assert.Nil(t, err)

	output := control.doSetTemperature("FOO", "17.0")
	if assert.Len(t, output, 1) {
		assert.Equal(t, "setting temperature in FOO to 17.0", output[0].Text)
	}

	output = control.doSetTemperature("FOO", "auto")
	if assert.Len(t, output, 1) {
		assert.Equal(t, "setting FOO back to auto", output[0].Text)
	}

	output = control.doSetTemperature("wrong room", "ABC")
	if assert.Len(t, output, 2) {
		assert.Equal(t, "invalid temperature ABC", output[0].Text)
		assert.Equal(t, "unknown room wrong room", output[1].Text)
	}
}
