package controller_test

import (
	"github.com/clambin/tado-exporter/internal/controller"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayConfigBlack(t *testing.T) {
	rules, err := controller.ParseRules([]byte(`
autoAway:
  - zoneName: "foo"
    mobileDeviceName: "foo"
    waitTime: 1h
    targetTemperature: 5.0
  - zoneName: "bar"
    mobileDeviceName: "bar"
    waitTime: 1h
    targetTemperature: 15.0
`))
	server := mockapi.MockAPI{}
	control := controller.Controller{
		API:           &server,
		Configuration: &controller.Configuration{},
		Rules:         rules,
		AutoAwayInfo:  nil,
	}

	assert.Nil(t, err)
	assert.Nil(t, control.AutoAwayInfo)

	err = control.Run()
	assert.Nil(t, err)
	assert.NotNil(t, control.AutoAwayInfo)
	assert.Len(t, control.AutoAwayInfo, 2)

	// "foo" was previously away
	autoAway, _ := control.AutoAwayInfo[1]
	autoAway.Home = false
	control.AutoAwayInfo[1] = autoAway

	err = control.Run()
	assert.Nil(t, err)
	assert.Len(t, server.Overlays, 0)

	// "bar" was previously home
	autoAway, _ = control.AutoAwayInfo[2]
	oldActivationTime := autoAway.ActivationTime
	autoAway.Home = true
	control.AutoAwayInfo[2] = autoAway

	err = control.Run()
	assert.Nil(t, err)

	autoAway, _ = control.AutoAwayInfo[2]
	assert.False(t, autoAway.Home)
	assert.True(t, autoAway.ActivationTime.After(oldActivationTime))

	// "bar" has been away for a long time
	autoAway, _ = control.AutoAwayInfo[2]
	autoAway.ActivationTime = time.Now().Add(-2 * time.Hour)
	control.AutoAwayInfo[2] = autoAway
	err = control.Run()
	assert.Nil(t, err)
	assert.Len(t, server.Overlays, 1)
	assert.Equal(t, 15.0, server.Overlays[2])
}
