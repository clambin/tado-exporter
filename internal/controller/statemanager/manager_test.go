package statemanager_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/statemanager"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var fakeUpdates = []poller.Update{
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneAuto, Temperature: tado.Temperature{Celsius: 25.0}}},
		UserStates: map[int]models.UserState{2: models.UserAway},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneOff, Temperature: tado.Temperature{Celsius: 15.0}}},
		UserStates: map[int]models.UserState{2: models.UserHome},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 20.0}}},
		UserStates: map[int]models.UserState{2: models.UserHome},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneAuto, Temperature: tado.Temperature{Celsius: 25.0}}},
		UserStates: map[int]models.UserState{2: models.UserHome},
	},
	{
		ZoneStates: map[int]models.ZoneState{2: {State: models.ZoneOff, Temperature: tado.Temperature{Celsius: 15.0}}},
		UserStates: map[int]models.UserState{2: models.UserAway},
	},
}

func TestManager_GetNextState_LimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   time.Hour,
		},
	}}
	mgr, err := statemanager.New(&mockapi.MockAPI{}, zoneConfig)
	assert.NoError(t, err)

	assert.True(t, mgr.IsValidZoneID(2))
	assert.False(t, mgr.IsValidZoneID(3))

	expectedResults := []struct {
		state  models.ZoneStateEnum
		delay  bool
		reason string
	}{
		{state: models.ZoneAuto, delay: false, reason: ""},
		{state: models.ZoneOff, delay: false, reason: ""},
		{state: models.ZoneAuto, delay: true, reason: "manual temperature setting detected in bar"},
	}

	for index, expectedResult := range expectedResults {
		nextState, when, reason := mgr.GetNextState(2, fakeUpdates[index])
		assert.Equal(t, expectedResult.state, nextState.State, index)
		assert.Equal(t, expectedResult.reason, reason, index)
		if expectedResult.delay {
			assert.NotZero(t, when)
		} else {
			assert.Zero(t, when)
		}
	}
}

func TestZoneManager_NightTime(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    23,
				Minutes: 30,
			},
		},
	}}
	mgr, err := statemanager.New(&mockapi.MockAPI{}, zoneConfig)
	assert.NoError(t, err)

	nextState, when, reason := mgr.GetNextState(2, fakeUpdates[2])
	assert.Equal(t, models.ZoneAuto, nextState.State)
	assert.NotZero(t, when)
	assert.Equal(t, "manual temperature setting detected in bar", reason)

	nextState, _, _ = mgr.GetNextState(2, fakeUpdates[1])
	assert.Equal(t, models.ZoneOff, nextState.State)
}

func TestZoneManager_AutoAway(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 2,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   10 * time.Minute,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
	}}

	mgr, err := statemanager.New(&mockapi.MockAPI{}, zoneConfig)
	assert.NoError(t, err)

	expectedResults := []struct {
		state  models.ZoneStateEnum
		delay  bool
		reason string
	}{
		{state: models.ZoneOff, delay: true, reason: "bar: bar is away"},
		{state: models.ZoneAuto, delay: false, reason: "bar: bar is home"},
		{state: models.ZoneManual, delay: false, reason: ""},
		{state: models.ZoneAuto, delay: false, reason: ""},
		{state: models.ZoneOff, delay: true, reason: "bar: bar is away"},
	}

	for index, expectedResult := range expectedResults {
		nextState, when, reason := mgr.GetNextState(2, fakeUpdates[index])
		assert.Equal(t, expectedResult.state, nextState.State, index)
		assert.Equal(t, expectedResult.reason, reason, index)
		if expectedResult.delay {
			assert.NotZero(t, when)
		} else {
			assert.Zero(t, when)
		}
	}
}

func TestZoneManager_Combined(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 2,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   10 * time.Millisecond,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   20 * time.Minute,
		},
		NightTime: configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    01,
				Minutes: 30,
				Seconds: 30,
			},
		},
	}}

	mgr, err := statemanager.New(&mockapi.MockAPI{}, zoneConfig)
	assert.NoError(t, err)

	expectedResults := []struct {
		state  models.ZoneStateEnum
		delay  bool
		reason string
	}{
		{state: models.ZoneOff, delay: true, reason: "bar: bar is away"},
		{state: models.ZoneAuto, delay: false, reason: "bar: bar is home"},
		{state: models.ZoneAuto, delay: true, reason: "manual temperature setting detected in bar"},
		{state: models.ZoneAuto, delay: false, reason: ""},
		{state: models.ZoneOff, delay: true, reason: "bar: bar is away"},
	}

	for index, expectedResult := range expectedResults {
		nextState, when, reason := mgr.GetNextState(2, fakeUpdates[index])
		assert.Equal(t, expectedResult.state, nextState.State, index)
		assert.Equal(t, expectedResult.reason, reason, index)
		if expectedResult.delay {
			assert.NotZero(t, when)
		} else {
			assert.Zero(t, when)
		}
	}
}
