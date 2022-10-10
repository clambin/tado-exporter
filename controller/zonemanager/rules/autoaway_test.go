package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAutoAwayRule_Evaluate(t *testing.T) {
	var testCases = []testCase{
		{
			name: "user goes away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZoneInfoSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateOff, Delay: time.Hour, ActionReason: "user(s) is/are away", CancelReason: "user(s) is/are home"},
		},
		{
			name: "user comes home",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 5.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateAuto, Delay: 0, ActionReason: "user(s) is/are home", CancelReason: "user(s) is/are away"},
		},
		{
			name: "user is home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZoneInfoSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: nil,
		},
		{
			name: "non-geolocation user",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 5.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: false}}},
			},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateAuto, Delay: 0, ActionReason: "user(s) is/are home", CancelReason: "user(s) is/are away"},
		},
	}

	r := &AutoAwayRule{
		zoneID:   10,
		zoneName: "living room",
		config: &configuration.ZoneAutoAway{
			Enabled: true,
			Delay:   time.Hour,
			Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := r.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}
