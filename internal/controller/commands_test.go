package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/tadoproxy"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestController_doRooms(t *testing.T) {
	cfg := &configuration.ControllerConfiguration{
		Enabled:    true,
		Interval:   1 * time.Minute,
		ZoneConfig: &[]configuration.ZoneConfig{},
	}

	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	ctrl := NewWithProxy(proxy, cfg)
	go ctrl.Run()

	ctrl.proxy.SetZones <- map[int]model.ZoneState{
		2: {State: model.Manual, Temperature: tado.Temperature{Celsius: 25.0}},
	}

	output := ctrl.doRooms()
	if assert.Len(t, output, 1) {
		lines := strings.Split(output[0].Text, "\n")
		if assert.Len(t, lines, 2) {
			assert.Equal(t, "bar: manual (25.0ºC)", lines[0])
			assert.Equal(t, "foo: auto", lines[1])
		}
	}

	ctrl.Stop()
}

func TestController_doUsers(t *testing.T) {
	cfg := &configuration.ControllerConfiguration{
		Enabled:    true,
		Interval:   1 * time.Minute,
		ZoneConfig: &[]configuration.ZoneConfig{},
	}

	proxy := tadoproxy.New("", "", "")
	proxy.API = &mockapi.MockAPI{}
	go proxy.Run()

	ctrl := NewWithProxy(proxy, cfg)
	go ctrl.Run()

	output := ctrl.doUsers()
	if assert.Len(t, output, 1) {
		lines := strings.Split(output[0].Text, "\n")
		if assert.Len(t, lines, 2) {
			assert.Equal(t, "bar: away", lines[0])
			assert.Equal(t, "foo: home", lines[1])
		}
	}

	ctrl.Stop()
}

/*
func TestController_doSetTemperature(t *testing.T) {
	ctrl := New("", "", "", nil)
	ctrl.API = &mockapi.MockAPI{}
	ctrl.roomSetter.API = &mockapi.MockAPI{}
	err := ctrl.Run()
	assert.Nil(t, err)

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
		control = New("", "", "", &cfg.Controller)
		control.API = &mockapi.MockAPI{}

		// log.SetLevel(log.DebugLevel)
		err = control.Run()
		assert.Nil(t, err)

		time.Sleep(500 * time.Millisecond)

		output := control.doRules()

		if assert.Len(t, output, 2) {
			assert.Equal(t, "bar is away. will set bar to manual in 1h0m0s\nfoo is home", output[0].Text)
			assert.Equal(t, "bar will be reset to auto in 1h0m0s", output[1].Text)
		}
	}
}


*/
