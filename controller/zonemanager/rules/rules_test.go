package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

type testCase struct {
	name        string
	update      *poller.Update
	targetState TargetState
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
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: poller.ZoneStateOff, Delay: time.Hour, Reason: "foo is away"},
		},
		{
			name: "user home - auto control",
			update: &poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: {Setting: tado.ZonePowerSetting{Power: "ON"}}},
				UserInfo: map[int]tado.MobileDevice{100: {ID: 100, Name: "foo", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
			},
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: false, State: poller.ZoneStateUnknown, Delay: 0, Reason: "foo is home, no manual settings detected"},
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
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: poller.ZoneStateAuto, Delay: 15 * time.Minute, Reason: "manual temp setting detected"},
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
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: poller.ZoneStateOff, Delay: time.Hour, Reason: "foo is away"},
		},
	}

	e := Evaluator{
		Config: &ZoneConfig{
			Zone: "living room",
			Rules: []RuleConfig{
				{Kind: LimitOverlay, Delay: 15 * time.Minute},
				{Kind: NightTime, Timestamp: Timestamp{Hour: 23, Minutes: 30}},
				{Kind: AutoAway, Delay: time.Hour, Users: []string{"foo"}},
			},
		},
	}

	testForceTime = time.Date(2022, 10, 10, 23, 0, 0, 0, time.Local)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := e.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.targetState, a)
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
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: false, State: poller.ZoneStateUnknown, Delay: 0, Reason: "no manual settings detected"},
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
			targetState: TargetState{ZoneID: 10, ZoneName: "living room", Action: true, State: poller.ZoneStateAuto, Delay: 30 * time.Minute, Reason: "manual temp setting detected"},
		},
	}

	e := Evaluator{
		Config: &ZoneConfig{
			Zone: "living room",
			Rules: []RuleConfig{
				{Kind: LimitOverlay, Delay: time.Hour},
				{Kind: NightTime, Timestamp: Timestamp{Hour: 23, Minutes: 30}},
			},
		},
	}

	testForceTime = time.Date(2022, 10, 10, 23, 0, 0, 0, time.Local)

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := e.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.targetState, a)
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

func TestNextState_LogValue(t *testing.T) {
	tests := []struct {
		name  string
		state poller.ZoneState
		delay time.Duration
		want  string
	}{
		{
			name:  "no overlay",
			state: poller.ZoneStateAuto,
			want:  "id=1, name=foo, state=auto, delay=0s, reason=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := TargetState{
				ZoneID:   1,
				ZoneName: "foo",
				State:    tt.state,
				Delay:    tt.delay,
			}

			var output []string
			for _, a := range s.LogValue().Group() {
				output = append(output, a.String())
			}
			assert.Equal(t, tt.want, strings.Join(output, ", "))
		})
	}
}
