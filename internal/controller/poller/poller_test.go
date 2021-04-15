package poller_test

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPoller_Run(t *testing.T) {
	p := poller.New(&mockapi.MockAPI{})

	var update poller.Update
	var err error

	update, err = p.Update()

	if assert.Nil(t, err) {
		if state, ok := update.ZoneStates[1]; assert.True(t, ok) {
			assert.Equal(t, models.ZoneAuto, state.State)
		}

		if state, ok := update.ZoneStates[2]; assert.True(t, ok) {
			assert.Equal(t, models.ZoneAuto, state.State)
		}

		if state, ok := update.UserStates[1]; assert.True(t, ok) {
			assert.Equal(t, models.UserHome, state)
		}

		if state, ok := update.UserStates[2]; assert.True(t, ok) {
			assert.Equal(t, models.UserAway, state)
		}
	}

	err = p.API.SetZoneOverlay(2, 18.5)
	if assert.Nil(t, err) == false {
		return
	}

	update, err = p.Update()

	if assert.Nil(t, err) {
		if state, ok := update.ZoneStates[2]; assert.True(t, ok) {
			assert.Equal(t, models.ZoneManual, state.State)
			assert.Equal(t, 18.5, state.Temperature.Celsius)
		}
	}

	err = p.API.SetZoneOverlay(2, 5.0)
	if assert.Nil(t, err) == false {
		return
	}

	update, err = p.Update()

	if assert.Nil(t, err) {
		if state, ok := update.ZoneStates[2]; assert.True(t, ok) {
			assert.Equal(t, models.ZoneOff, state.State)
		}
	}
}
