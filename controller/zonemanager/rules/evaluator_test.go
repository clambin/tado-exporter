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

type testCase struct {
	name   string
	update *poller.Update
	action *NextState
}

func TestEvaluator_Evaluate(t *testing.T) {
	var testCases = []testCase{
		{
			name: "user away - auto control",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZoneInfoSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateOff, Delay: time.Hour, ActionReason: "foo is away", CancelReason: "foo is home"},
		},
		{
			name: "user home - auto control",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZoneInfoSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: nil,
		},
		{
			name: "user home - manual control",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateAuto, Delay: 15 * time.Minute, ActionReason: "manual temp setting detected", CancelReason: "room no longer in manual temp setting"},
		},
		{
			name: "user away - manual control",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateOff, Delay: time.Hour, ActionReason: "foo is away", CancelReason: "foo is home"},
		},
	}

	e := &Evaluator{
		Config: &configuration.ZoneConfig{
			ZoneID: 10,
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Delay:   time.Hour,
				Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}},
			},
			LimitOverlay: configuration.ZoneLimitOverlay{
				Enabled: true,
				Delay:   15 * time.Minute,
			},
			NightTime: configuration.ZoneNightTime{
				Enabled: true,
				Time: configuration.ZoneNightTimeTimestamp{
					Hour:    23,
					Minutes: 30,
				},
			},
		},
	}

	testForceTime = time.Date(2022, 10, 10, 23, 0, 0, 0, time.Local)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := e.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}

func TestEvaluator_Evaluate_LimitOverlay_Vs_NightTime(t *testing.T) {
	var testCases = []testCase{
		{
			name: "user home - auto control",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZoneInfoSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: nil,
		},
		{
			name: "user home - manual control",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateAuto, Delay: 30 * time.Minute, ActionReason: "manual temp setting detected", CancelReason: "room no longer in manual temp setting"},
		},
	}

	e := &Evaluator{
		Config: &configuration.ZoneConfig{
			ZoneID: 10,
			LimitOverlay: configuration.ZoneLimitOverlay{
				Enabled: true,
				Delay:   time.Hour,
			},
			NightTime: configuration.ZoneNightTime{
				Enabled: true,
				Time: configuration.ZoneNightTimeTimestamp{
					Hour:    23,
					Minutes: 30,
				},
			},
		},
	}

	testForceTime = time.Date(2022, 10, 10, 23, 0, 0, 0, time.Local)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := e.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}

func TestEvaluator_Evaluate_BadConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config *configuration.ZoneConfig
	}{
		{
			name: "bad zone name",
			config: &configuration.ZoneConfig{
				ZoneName:     "foo",
				LimitOverlay: configuration.ZoneLimitOverlay{Enabled: true, Delay: time.Hour},
			},
		},
		{
			name: "bad user name",
			config: &configuration.ZoneConfig{
				ZoneID:   10,
				AutoAway: configuration.ZoneAutoAway{Enabled: true, Delay: time.Hour, Users: []configuration.ZoneUser{{MobileDeviceName: "bar"}}},
			},
		},
	}

	var update = &poller.Update{
		Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZoneInfoSetting{Power: "ON"}}},
		UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			e := &Evaluator{Config: tt.config}
			_, err := e.Evaluate(update)
			assert.Error(t, err)
		})
	}
}

func BenchmarkEvaluator(b *testing.B) {
	update := &poller.Update{
		Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	e := &Evaluator{
		Config: &configuration.ZoneConfig{
			ZoneID: 10,
			AutoAway: configuration.ZoneAutoAway{
				Enabled: true,
				Delay:   time.Hour,
				Users:   []configuration.ZoneUser{{MobileDeviceName: "foo"}},
			},
			LimitOverlay: configuration.ZoneLimitOverlay{
				Enabled: true,
				Delay:   15 * time.Minute,
			},
			NightTime: configuration.ZoneNightTime{
				Enabled: true,
				Time: configuration.ZoneNightTimeTimestamp{
					Hour:    23,
					Minutes: 30,
				},
			},
		},
	}
	for i := 0; i < b.N; i++ {
		if _, err := e.Evaluate(update); err != nil {
			b.Fatal(err)
		}
	}
}
