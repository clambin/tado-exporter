package processor

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/setter"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var (
	zoneConfig = []configuration.ZoneConfig{
		{
			ZoneName: "foo",
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Delay:   2 * time.Hour,
				Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}, {MobileDeviceName: "bar"}},
			},
			LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: time.Hour},
			NightTime:    configuration.ZoneNightTime{Enabled: true, Time: configuration.ZoneNightTimeTimestamp{Hour: 23, Minutes: 30}},
		},
		{
			ZoneName:     "bar",
			LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: time.Hour},
		},
	}

	testUpdates = []poller.Update{
		{
			Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
			ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
			UserInfo: map[int]tado.MobileDevice{1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
		},
		{
			Zones:    map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
			ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
			UserInfo: map[int]tado.MobileDevice{1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
		},
		{
			Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
			ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}}},
			UserInfo: map[int]tado.MobileDevice{1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
		},
		{
			Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
			ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}}},
			UserInfo: map[int]tado.MobileDevice{1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
		},
		{
			Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
			ZoneInfo: map[int]tado.ZoneInfo{1: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}}},
			UserInfo: map[int]tado.MobileDevice{1: {ID: 1, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
		},
	}
)

func TestServer_Load(t *testing.T) {
	p := New(zoneConfig)
	require.NotNil(t, p)
	p.load(&testUpdates[0])
	require.Len(t, p.zoneRules, 1)
	rule, ok := p.zoneRules[1]
	require.True(t, ok)

	assert.True(t, rule.LimitOverlay.Enabled)
	assert.Equal(t, time.Hour, rule.LimitOverlay.Delay)

	assert.True(t, rule.AutoAway.Enabled)
	assert.Equal(t, []int{1}, rule.AutoAway.Users)

	assert.True(t, rule.NightTime.Enabled)
	assert.Equal(t, 23, rule.NightTime.Time.Hour)
	assert.Equal(t, 30, rule.NightTime.Time.Minutes)
}

func TestServer_GetNextState(t *testing.T) {
	p := New(zoneConfig)
	require.NotNil(t, p)
	p.load(&testUpdates[0])

	// User is home
	_, nextState, delay, reason, err := p.getNextState(1, &testUpdates[0])
	require.NoError(t, err)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), nextState)
	assert.Zero(t, delay)
	assert.Empty(t, reason)

	// User is not home: autoAway triggers
	_, nextState, delay, reason, err = p.getNextState(1, &testUpdates[1])
	require.NoError(t, err)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateOff), nextState)
	assert.Equal(t, 2*time.Hour, delay)
	assert.Equal(t, "foo: foo is away", reason)

	// Zone is in overlay mode: limitOverlay or nightTime triggers
	_, nextState, delay, reason, err = p.getNextState(1, &testUpdates[2])
	require.NoError(t, err)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateAuto), nextState)
	assert.LessOrEqual(t, delay, time.Hour)
	// assert.NotZero(t, delay)
	assert.Equal(t, "manual temperature setting detected in foo", reason)
}

func TestServer_Process(t *testing.T) {
	zoneConfig2 := []configuration.ZoneConfig{
		{
			ZoneName: "foo",
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Delay:   2 * time.Hour,
				Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}, {MobileDeviceName: "bar"}},
			},
			LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: time.Hour},
		},
		{
			ZoneName:     "bar",
			LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: time.Hour},
		},
	}

	p := New(zoneConfig2)
	require.NotNil(t, p)

	// User is home, zone in Auto. No action
	nextStates := p.Process(&testUpdates[0])
	require.Len(t, nextStates, 1)
	assert.Nil(t, nextStates[1])

	// User is away. Zone should be switched off
	nextStates = p.Process(&testUpdates[1])
	require.Len(t, nextStates, 1)
	require.NotNil(t, nextStates[1])
	assert.Equal(t, setter.NextState{State: tado.ZoneStateOff, Delay: 2 * time.Hour, Reason: "foo: foo is away"}, *nextStates[1])

	// User is home, zone is in manual mode. Zone should be switched to auto mode
	nextStates = p.Process(&testUpdates[2])
	require.Len(t, nextStates, 1)
	require.NotNil(t, nextStates[1])
	assert.Equal(t, setter.NextState{State: tado.ZoneStateAuto, Delay: time.Hour, Reason: "manual temperature setting detected in foo"}, *nextStates[1])

	// User is home, zone is off. Zone should be switched to auto mode
	nextStates = p.Process(&testUpdates[3])
	require.Len(t, nextStates, 1)
	require.NotNil(t, nextStates[1])
	assert.Equal(t, setter.NextState{State: tado.ZoneStateAuto, Delay: 0, Reason: "foo: foo is home"}, *nextStates[1])

	// User is away, zone is off. No action
	nextStates = p.Process(&testUpdates[4])
	require.Len(t, nextStates, 1)
	require.Nil(t, nextStates[1])
}
