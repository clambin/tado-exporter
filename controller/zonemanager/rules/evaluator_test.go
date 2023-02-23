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

type testCase struct {
	name   string
	update *poller.Update
	action NextState
}

func TestEvaluator_Evaluate(t *testing.T) {
	var testCases = []testCase{
		{
			name: "user away - auto control",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			action: NextState{ZoneID: 10, ZoneName: "living room", State: tado2.ZoneStateOff, Delay: time.Hour, ActionReason: "foo is away", CancelReason: "foo is home"},
		},
		{
			name: "user home - auto control",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			//action: nil,
		},
		{
			name: "user home - manual control",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZonePowerSetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: NextState{ZoneID: 10, ZoneName: "living room", State: tado2.ZoneStateAuto, Delay: 15 * time.Minute, ActionReason: "manual temp setting detected", CancelReason: "room no longer in manual temp setting"},
		},
		{
			name: "user away - manual control",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZonePowerSetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
			},
			action: NextState{ZoneID: 10, ZoneName: "living room", State: tado2.ZoneStateOff, Delay: time.Hour, ActionReason: "foo is away", CancelReason: "foo is home"},
		},
	}

	e := Evaluator{
		Config: &ZoneConfig{
			Zone: "living room",
			Rules: []RuleConfig{
				{
					Kind:  LimitOverlay,
					Delay: 15 * time.Minute,
				},
				{
					Kind:      NightTime,
					Timestamp: Timestamp{Hour: 23, Minutes: 30},
				},
				{
					Kind:  AutoAway,
					Delay: time.Hour,
					Users: []string{"foo"},
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
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			//action: nil,
		},
		{
			name: "user home - manual control",
			update: &poller.Update{
				Zones: map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
					Type:        "MANUAL",
					Setting:     tado.ZonePowerSetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
					Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
				}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			action: NextState{ZoneID: 10, ZoneName: "living room", State: tado2.ZoneStateAuto, Delay: 30 * time.Minute, ActionReason: "manual temp setting detected", CancelReason: "room no longer in manual temp setting"},
		},
	}

	e := Evaluator{
		Config: &ZoneConfig{
			Zone: "living room",
			Rules: []RuleConfig{
				{
					Kind:  LimitOverlay,
					Delay: time.Hour,
				},
				{
					Kind:      NightTime,
					Timestamp: Timestamp{Hour: 23, Minutes: 30},
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
		config ZoneConfig
	}{
		{
			name: "limitOverlay - bad zone name",
			config: ZoneConfig{
				Zone: "foo",
				Rules: []RuleConfig{
					{
						Kind:  LimitOverlay,
						Delay: time.Hour,
					},
				},
			},
		},
		{
			name: "autoAway - bad zone name",
			config: ZoneConfig{
				Zone: "foo",
				Rules: []RuleConfig{
					{
						Kind:  AutoAway,
						Delay: time.Hour,
						Users: []string{"foo"},
					},
				},
			},
		},
		{
			name: "autoAway - bad user name",
			config: ZoneConfig{
				Zone: "living room",
				Rules: []RuleConfig{
					{
						Kind:  AutoAway,
						Delay: time.Hour,
						Users: []string{"bar"},
					},
				},
			},
		},
	}

	var update = &poller.Update{
		Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
		UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			e := Evaluator{Config: &tt.config}
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
			Setting:     tado.ZonePowerSetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	}

	e := Evaluator{
		Config: &ZoneConfig{
			Zone: "living room",
			Rules: []RuleConfig{
				{
					Kind:  AutoAway,
					Delay: time.Hour,
					Users: []string{"foo"},
				},
				{
					Kind:  LimitOverlay,
					Delay: 15 * time.Minute,
				},
				{
					Kind:      NightTime,
					Timestamp: Timestamp{Hour: 23, Minutes: 30},
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
