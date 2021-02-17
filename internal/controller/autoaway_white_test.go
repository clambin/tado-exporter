package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAutoAwayConfigWhite(t *testing.T) {
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
  - zoneName: "bar"
    mobileDeviceName: "not-a-phone"
    waitTime: 1h
    targetTemperature: 7.0
  - zoneName: "not-a-zone"
    mobileDeviceName: "foo"
    waitTime: 1h
    targetTemperature: 7.0
`))
	if err != nil {
		panic(err)
	}

	ctrlr := Controller{
		API:           &mockapi.MockAPI{},
		Configuration: &cfg.Controller,
	}

	assert.Nil(t, err)
	assert.Nil(t, ctrlr.AutoAwayInfo)

	err = ctrlr.updateTadoConfig()
	assert.Nil(t, err)

	err = ctrlr.updateAutoAwayInfo()
	assert.Nil(t, err)
	assert.NotNil(t, ctrlr.AutoAwayInfo)
	assert.Len(t, ctrlr.AutoAwayInfo, 2)

	actions, err := ctrlr.getAutoAwayActions()
	assert.Nil(t, err)
	assert.Len(t, actions, 0)

	// "foo" was previously away
	autoAway, _ := ctrlr.AutoAwayInfo[1]
	autoAway.state = autoAwayStateAway
	ctrlr.AutoAwayInfo[1] = autoAway

	err = ctrlr.updateAutoAwayInfo()
	assert.Nil(t, err)
	actions, err = ctrlr.getAutoAwayActions()
	assert.Nil(t, err)
	assert.Len(t, actions, 1)
	// "foo" now home, so we need to delete the overlay
	assert.False(t, actions[0].Overlay)
	assert.Equal(t, 1, actions[0].ZoneID)

	// "bar" was previously home
	autoAway, _ = ctrlr.AutoAwayInfo[2]
	autoAway.ActivationTime = time.Now().Add(-2 * time.Hour)
	ctrlr.AutoAwayInfo[2] = autoAway

	err = ctrlr.updateAutoAwayInfo()
	assert.Nil(t, err)
	actions, err = ctrlr.getAutoAwayActions()
	assert.Nil(t, err)
	assert.Len(t, actions, 1)
	// "bar" has been away longer than WaitTime, so we need to set an overlay
	assert.True(t, actions[0].Overlay)
	assert.Equal(t, 2, actions[0].ZoneID)
	assert.Equal(t, 15.0, actions[0].TargetTemperature)
}
