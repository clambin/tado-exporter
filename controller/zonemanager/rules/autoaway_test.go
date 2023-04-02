package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAutoAwayRule_Evaluate(t *testing.T) {
	tests := []testCase{
		{
			name: "user goes away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 18.0}}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "foo is away"},
		},
		{
			name: "user is away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 5.0}}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "foo is away"},
		},
		{
			name: "user comes home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 5.0}}, Overlay: tado.ZoneInfoOverlay{Type: "HEATING", Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"}}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Delay: 0, Reason: "foo is home"},
		},
		{
			name: "user is home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 18.0}}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "foo is home"},
		},
		{
			name: "non-geolocation user",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 15.0}}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: false}}},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: ""},
		},
	}

	r := &AutoAwayRule{
		ZoneID:   10,
		ZoneName: "living room",
		Delay:    time.Hour,
		Users:    []string{"foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := r.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}

func TestAutoAwayRule_Evaluate_MultipleUsers(t *testing.T) {
	tests := []testCase{
		{
			name: "one user goes away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 18.0}}}},
				UserInfo: map[int]tado.MobileDevice{
					100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
					110: {ID: 100, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
				},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "bar is home"},
		},
		{
			name: "all users are away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 18.0}}}},
				UserInfo: map[int]tado.MobileDevice{
					100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
					110: {ID: 100, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
				},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "foo, bar are away"},
		},
		{
			name: "one user is home",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {
					Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 5.0}},
					Overlay: tado.ZoneInfoOverlay{Type: "HEATING", Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"}},
				}},
				UserInfo: map[int]tado.MobileDevice{
					100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
					110: {ID: 100, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
				},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Reason: "bar is home"},
		},
		{
			name: "user is home, schedule for room is off",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {
					Setting: tado.ZonePowerSetting{Temperature: tado.Temperature{Celsius: 5.0}}},
				},
				UserInfo: map[int]tado.MobileDevice{
					100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
					110: {ID: 100, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
				},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "bar is home"},
		},
	}

	r := &AutoAwayRule{
		ZoneID:   10,
		ZoneName: "living room",
		Delay:    time.Hour,
		Users:    []string{"foo", "bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := r.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}
