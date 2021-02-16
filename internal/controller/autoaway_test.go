package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayConfigBlack(t *testing.T) {
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

	if err != nil {
		panic(err)
	}

	server := mockapi.MockAPI{}
	control := Controller{
		API:           &server,
		Configuration: &cfg.Controller,
	}

	assert.Nil(t, err)
	assert.Nil(t, control.AutoAwayInfo)

	err = control.Run()
	assert.Nil(t, err)
	assert.NotNil(t, control.AutoAwayInfo)
	assert.Len(t, control.AutoAwayInfo, 2)

	// "foo" was previously away
	autoAway, _ := control.AutoAwayInfo[1]
	autoAway.state = autoAwayStateAway
	control.AutoAwayInfo[1] = autoAway

	err = control.Run()
	assert.Nil(t, err)
	assert.Len(t, server.Overlays, 0)

	// "bar" was previously home
	autoAway, _ = control.AutoAwayInfo[2]
	oldActivationTime := autoAway.ActivationTime
	autoAway.state = autoAwayStateHome
	control.AutoAwayInfo[2] = autoAway

	err = control.Run()
	assert.Nil(t, err)

	autoAway, _ = control.AutoAwayInfo[2]
	assert.Equal(t, autoAwayState(autoAwayStateAway), autoAway.state)
	assert.True(t, autoAway.ActivationTime.After(oldActivationTime))

	// "bar" has been away for a long time
	autoAway, _ = control.AutoAwayInfo[2]
	autoAway.ActivationTime = time.Now().Add(-2 * time.Hour)
	control.AutoAwayInfo[2] = autoAway
	err = control.Run()
	assert.Nil(t, err)
	autoAway, _ = control.AutoAwayInfo[2]
	assert.Equal(t, autoAwayState(autoAwayStateReported), autoAway.state)
	assert.Len(t, server.Overlays, 1)
	assert.Equal(t, 15.0, server.Overlays[2])

	// TODO: once bar is away, it shouldn't trigger any more events
}
