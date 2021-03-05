package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestController_doRooms(t *testing.T) {
	ctrl, err := New("", "", "", nil)
	if assert.Nil(t, err) && assert.NotNil(t, ctrl) {

		ctrl.API = &mockapi.MockAPI{}

		err = ctrl.Run()
		assert.Nil(t, err)

		output := ctrl.doRooms()
		if assert.Len(t, output, 1) {
			lines := strings.Split(output[0].Text, "\n")
			if assert.Len(t, lines, 2) {
				assert.Equal(t, "bar: 19.9ºC (target: 25.0ºC MANUAL)", lines[0])
				assert.Equal(t, "foo: 19.9ºC (target: 20.0ºC MANUAL)", lines[1])
			}
		}
	}
}

func TestController_doUsers(t *testing.T) {
	ctrl, err := New("", "", "", nil)
	if assert.Nil(t, err) && assert.NotNil(t, ctrl) {

		ctrl.API = &mockapi.MockAPI{}

		err = ctrl.Run()
		assert.Nil(t, err)

		output := ctrl.doUsers()
		if assert.Len(t, output, 1) {
			lines := strings.Split(output[0].Text, "\n")
			if assert.Len(t, lines, 2) {
				assert.Equal(t, "bar: away", lines[0])
				assert.Equal(t, "foo: home", lines[1])
			}
		}
	}
}

func TestController_doSetTemperature(t *testing.T) {
	ctrl, err := New("", "", "", nil)
	if assert.Nil(t, err) && assert.NotNil(t, ctrl) {
		ctrl.API = &mockapi.MockAPI{}
		ctrl.roomSetter.API = &mockapi.MockAPI{}
		err = ctrl.Run()
		assert.Nil(t, err)
	}

	output := ctrl.doSetTemperature("bar", "auto")
	assert.Len(t, output, 1)
	assert.Equal(t, "setting bar back to auto", output[0].Text)

	output = ctrl.doSetTemperature("bar", "15.5")
	assert.Len(t, output, 1)
	assert.Equal(t, "setting temperature in bar to 15.5", output[0].Text)

	output = ctrl.doSetTemperature("bar", "15,5")
	assert.Len(t, output, 1)
	assert.Equal(t, "invalid temperature: 15,5", output[0].Text)

	output = ctrl.doSetTemperature("snafu", "auto")
	assert.Len(t, output, 1)
	assert.Equal(t, "unknown room name: snafu", output[0].Text)

	output = ctrl.doSetTemperature("auto")
	assert.Len(t, output, 1)
	assert.Equal(t, "invalid command:  set <room name> auto|<temperature>", output[0].Text)
}

func TestController_doRules(t *testing.T) {
	var (
		err     error
		cfg     *configuration.Configuration
		control *Controller
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
	if assert.Nil(t, err); err == nil {
		control, err = New("", "", "", &cfg.Controller)
	}

	if assert.Nil(t, err) && assert.NotNil(t, control) {
		control.API = &mockapi.MockAPI{}

		//		log.SetLevel(log.DebugLevel)
		err = control.Run()
		assert.Nil(t, err)

		output := control.doRules()

		if assert.Len(t, output, 1) {
			assert.Equal(t, "bar will be reset to auto in 1h0m0s", output[0].Text)
		}

		/*
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
		*/
	}
}
