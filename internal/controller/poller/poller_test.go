package poller_test

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPoller_Run(t *testing.T) {
	p := poller.New(&mockapi.MockAPI{}, 50*time.Millisecond)

	go p.Run()

	var update poller.Update
	assert.Eventually(t, func() bool {
		update = <-p.Update

		return len(update.ZoneStates) == 2 && len(update.UserStates) == 2
	}, 100*time.Millisecond, 50*time.Millisecond)

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

	err := p.API.SetZoneOverlay(2, 18.5)
	if assert.Nil(t, err) == false {
		return
	}

	// drain any queued updates (not thread-safe, but we have no concurrent readers)
	for len(p.Update) > 0 {
		<-p.Update
	}

	update = <-p.Update

	if state, ok := update.ZoneStates[2]; assert.True(t, ok) {
		assert.Equal(t, models.ZoneManual, state.State)
		assert.Equal(t, 18.5, state.Temperature.Celsius)
	}

	err = p.API.SetZoneOverlay(2, 5.0)
	if assert.Nil(t, err) == false {
		return
	}

	// drain any queued updates (not thread-safe, but we have no concurrent readers)
	for len(p.Update) > 0 {
		<-p.Update
	}

	update = <-p.Update

	if state, ok := update.ZoneStates[2]; assert.True(t, ok) {
		assert.Equal(t, models.ZoneOff, state.State)
	}

	p.Cancel <- struct{}{}

	assert.Eventually(t, func() bool {
		_, ok := <-p.Cancel
		return !ok
	}, 500*time.Millisecond, 10*time.Millisecond)
}
