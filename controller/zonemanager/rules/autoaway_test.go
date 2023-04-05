package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado/testutil"
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
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "foo is away"},
		},
		{
			name: "user is away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 5), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "foo is away"},
		},
		{
			name: "user comes home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 5), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Delay: 0, Reason: "foo is home"},
		},
		{
			name: "user is home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 19))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "foo is home"},
		},
		{
			name: "non-geolocation user",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo")},
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
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false)),
					110: testutil.MakeMobileDevice(110, "bar", testutil.Home(true)),
				},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "bar is home"},
		},
		{
			name: "all users are away",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false)),
					110: testutil.MakeMobileDevice(110, "bar", testutil.Home(false)),
				},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "foo, bar are away"},
		},
		{
			name: "one user is home",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 5), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false)),
					110: testutil.MakeMobileDevice(110, "bar", testutil.Home(true)),
				},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Reason: "bar is home"},
		},
		{
			name: "user is home, schedule for room is off",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false)),
					110: testutil.MakeMobileDevice(110, "bar", testutil.Home(true)),
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
