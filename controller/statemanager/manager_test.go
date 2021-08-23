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

var FakeUpdates = []poller.Update{
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
	},
}

func TestZoneManager_GetNextState_LimitOverlay(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneName: "bar",
		LimitOverlay: configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   time.Hour,
		},
	}}
	tadoCache := &cache.Cache{}
	mgr := statemanager.Manager{ZoneConfig: zoneConfig, Cache: tadoCache}

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
		tadoCache.Update(&FakeUpdates[index])
		nextState, when, reason, err := mgr.GetNextState(2, &FakeUpdates[index])
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

func TestZoneManager_GetNextState_NightTime(t *testing.T) {
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
	tadoCache := &cache.Cache{}
	mgr := statemanager.Manager{
		ZoneConfig: zoneConfig,
		Cache:      tadoCache,
	}

	tadoCache.Update(&FakeUpdates[2])
	nextState, when, reason, err := mgr.GetNextState(2, &FakeUpdates[2])
	assert.NoError(t, err)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), nextState)
	assert.NotZero(t, when)
	assert.Equal(t, "manual temperature setting detected in bar", reason)

	tadoCache.Update(&FakeUpdates[1])
	nextState, _, _, _ = mgr.GetNextState(2, &FakeUpdates[1])
	assert.Equal(t, tado.ZoneState(tado.ZoneStateOff), nextState)
}

func TestZoneManager_GetNextState_AutoAway(t *testing.T) {
	zoneConfig := []configuration.ZoneConfig{{
		ZoneID: 2,
		AutoAway: configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   10 * time.Minute,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "bar"}},
		},
	}}

	tadoCache := &cache.Cache{}
	mgr := statemanager.Manager{ZoneConfig: zoneConfig, Cache: tadoCache}

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
		tadoCache.Update(&FakeUpdates[index])
		nextState, when, reason, err := mgr.GetNextState(2, &FakeUpdates[index])
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

func TestZoneManager_GetNextState_Combined(t *testing.T) {
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

	tadoCache := &cache.Cache{}
	mgr := statemanager.Manager{ZoneConfig: zoneConfig, Cache: tadoCache}

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
		tadoCache.Update(&FakeUpdates[index])
		nextState, when, reason, err := mgr.GetNextState(2, &FakeUpdates[index])
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
