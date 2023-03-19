package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	tado2 "github.com/clambin/tado-exporter/tado"
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
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: tado2.ZoneStateOff, Delay: time.Hour, Reason: "foo is away"},
		},
		{
			name: "user is away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "OFF"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: false, State: tado2.ZoneStateUnknown, Delay: 0, Reason: "foo is away"},
		},
		{
			name: "user comes home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "OFF"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: tado2.ZoneStateAuto, Delay: 0, Reason: "foo is home"},
		},
		{
			name: "user is home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: false, State: tado2.ZoneStateUnknown, Delay: 0, Reason: "foo is home"},
		},
		{
			name: "non-geolocation user",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: false}}},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: false, State: tado2.ZoneStateUnknown, Delay: 0, Reason: ""},
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
			assert.Equal(t, tt.targetState, a)
		})
	}
}

func TestAutoAwayRule_Evaluate_MultipleUsers(t *testing.T) {
	tests := []testCase{
		{
			name: "one user goes away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{
					100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
					110: {ID: 100, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
				},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: false, State: tado2.ZoneStateUnknown, Delay: 0, Reason: "bar is home"},
		},
		{
			name: "all users are away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{
					100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
					110: {ID: 100, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
				},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: tado2.ZoneStateOff, Delay: time.Hour, Reason: "foo, bar are away"},
		},
		{
			name: "one user is home",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {
					Setting: tado.ZonePowerSetting{Power: "OFF"},
					Overlay: tado.ZoneInfoOverlay{
						Type:        "MANUAL",
						Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
					}}},
				UserInfo: map[int]tado.MobileDevice{
					100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}},
					110: {ID: 100, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}},
				},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: tado2.ZoneStateAuto, Delay: 0, Reason: "bar is home"},
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
			assert.Equal(t, tt.targetState, a)
		})
	}
}
