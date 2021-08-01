package statemanager_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/cache"
	"github.com/clambin/tado-exporter/controller/statemanager"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var fakeUpdates = []poller.Update{
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 15.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
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
	testcache := cache.New()
	mgr, err := statemanager.New(zoneConfig, testcache)
	assert.NoError(t, err)

	expectedResults := []struct {
		state  tado.ZoneState
		delay  bool
		reason string
	}{
		{state: tado.ZoneStateAuto, delay: false, reason: ""},
		{state: tado.ZoneStateOff, delay: false, reason: ""},
		{state: tado.ZoneStateAuto, delay: true, reason: "manual temperature setting detected in bar"},
	}

	for index, expectedResult := range expectedResults {
		var (
			nextState tado.ZoneState
			when      time.Duration
			reason    string
		)
		testcache.Update(&fakeUpdates[index])
		nextState, when, reason, err = mgr.GetNextState(2, &fakeUpdates[index])
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.state, nextState, index)
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
	testCache := cache.New()
	mgr, err := statemanager.New(zoneConfig, testCache)
	assert.NoError(t, err)

	var (
		nextState tado.ZoneState
		when      time.Duration
		reason    string
	)
	testCache.Update(&fakeUpdates[2])
	nextState, when, reason, err = mgr.GetNextState(2, &fakeUpdates[2])
	assert.NoError(t, err)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), nextState)
	assert.NotZero(t, when)
	assert.Equal(t, "manual temperature setting detected in bar", reason)

	testCache.Update(&fakeUpdates[1])
	nextState, _, _, _ = mgr.GetNextState(2, &fakeUpdates[1])
	assert.Equal(t, tado.ZoneState(tado.ZoneStateOff), nextState)
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

	testCache := cache.New()
	mgr, err := statemanager.New(zoneConfig, testCache)
	assert.NoError(t, err)

	expectedResults := []struct {
		state  tado.ZoneState
		delay  bool
		reason string
	}{
		{state: tado.ZoneStateOff, delay: true, reason: "bar: bar is away"},
		{state: tado.ZoneStateAuto, delay: false, reason: "bar: bar is home"},
		{state: tado.ZoneStateManual, delay: false, reason: ""},
		{state: tado.ZoneStateAuto, delay: false, reason: ""},
		{state: tado.ZoneStateOff, delay: true, reason: "bar: bar is away"},
	}

	for index, expectedResult := range expectedResults {
		var (
			nextState tado.ZoneState
			when      time.Duration
			reason    string
		)
		testCache.Update(&fakeUpdates[index])
		nextState, when, reason, err = mgr.GetNextState(2, &fakeUpdates[index])
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.state, nextState, index)
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

	testCache := cache.New()
	mgr, err := statemanager.New(zoneConfig, testCache)
	assert.NoError(t, err)

	expectedResults := []struct {
		state  tado.ZoneState
		delay  bool
		reason string
	}{
		{state: tado.ZoneStateOff, delay: true, reason: "bar: bar is away"},
		{state: tado.ZoneStateAuto, delay: false, reason: "bar: bar is home"},
		{state: tado.ZoneStateAuto, delay: true, reason: "manual temperature setting detected in bar"},
		{state: tado.ZoneStateAuto, delay: false, reason: ""},
		{state: tado.ZoneStateOff, delay: true, reason: "bar: bar is away"},
	}

	for index, expectedResult := range expectedResults {
		var (
			nextState tado.ZoneState
			when      time.Duration
			reason    string
		)
		testCache.Update(&fakeUpdates[index])
		nextState, when, reason, err = mgr.GetNextState(2, &fakeUpdates[index])
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.state, nextState, index)
		assert.Equal(t, expectedResult.reason, reason, index)
		if expectedResult.delay {
			assert.NotZero(t, when)
		} else {
			assert.Zero(t, when)
		}
	}
}
