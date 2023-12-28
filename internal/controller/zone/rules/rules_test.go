package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type testCase struct {
	name   string
	update poller.Update
	action Action
}

func TestEvaluator_Evaluate(t *testing.T) {
	var testCases = []testCase{
		{
			name: "user away - auto control",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "foo is away"},
		},
		{
			name: "user home - auto control",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "foo is home, no manual settings detected"},
		},
		{
			name: "user home - manual control",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Delay: 15 * time.Minute, Reason: "manual temp setting detected"},
		},
		{
			name: "user away - manual control",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "foo is away"},
		},
		{
			name: "home in away mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTadoMode(false), testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(false))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "device in away mode"},
		},
	}

	testForceTime = time.Date(2022, 10, 10, 23, 0, 0, 0, time.Local)

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := Evaluator{
				Config: &ZoneConfig{
					Zone: "living room",
					Rules: []ZoneRule{
						{Kind: LimitOverlay, Delay: 15 * time.Minute},
						{Kind: NightTime, Timestamp: Timestamp{Hour: 23, Minutes: 30}},
						{Kind: AutoAway, Delay: time.Hour, Users: []string{"foo"}},
					},
				},
			}

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
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "no manual settings detected"},
		},
		{
			name: "user home - manual control",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())},
				UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
			},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Delay: 30 * time.Minute, Reason: "manual temp setting detected"},
		},
	}

	e := Evaluator{
		Config: &ZoneConfig{
			Zone: "living room",
			Rules: []ZoneRule{
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
				Rules: []ZoneRule{
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
				Rules: []ZoneRule{
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
				Rules: []ZoneRule{
					{
						Kind:  AutoAway,
						Delay: time.Hour,
						Users: []string{"bar"},
					},
				},
			},
		},
	}

	update := poller.Update{
		Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18))},
		UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
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
	update := poller.Update{
		Zones:    map[int]tado.Zone{10: {ID: 10, Name: "living room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())},
		UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "foo", testutil.Home(true))},
	}

	e := Evaluator{
		Config: &ZoneConfig{
			Zone: "living room",
			Rules: []ZoneRule{
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

func TestEvaluator_zoneInHomeMode(t *testing.T) {
	testCases := []struct {
		name    string
		update  poller.Update
		wantID  int
		wantOK  assert.BoolAssertionFunc
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "home",
			update: poller.Update{
				Zones: map[int]tado.Zone{
					1: {ID: 1, Name: "room"},
				},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo()},
				Home:     true,
			},
			wantID:  1,
			wantOK:  assert.True,
			wantErr: assert.NoError,
		},
		{
			name: "away",
			update: poller.Update{
				Zones: map[int]tado.Zone{
					1: {ID: 1, Name: "room"},
				},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo(testutil.ZoneInfoTadoMode(false))},
				Home:     false,
			},
			wantID:  1,
			wantOK:  assert.False,
			wantErr: assert.NoError,
		},
		{
			name: "invalid room",
			update: poller.Update{
				Zones: map[int]tado.Zone{
					1: {ID: 1, Name: "other room"},
				},
				ZoneInfo: map[int]tado.ZoneInfo{1: testutil.MakeZoneInfo()},
				Home:     true,
			},
			wantOK:  assert.False,
			wantErr: assert.Error,
		},
		{
			name: "invalid update",
			update: poller.Update{
				Zones: map[int]tado.Zone{
					1: {ID: 1, Name: "room"},
				},
				ZoneInfo: map[int]tado.ZoneInfo{2: testutil.MakeZoneInfo()},
				Home:     true,
			},
			wantOK:  assert.False,
			wantErr: assert.Error,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := Evaluator{Config: &ZoneConfig{Zone: "room"}}
			id, ok, err := e.zoneInHomeMode(tt.update)
			assert.Equal(t, tt.wantID, id)
			tt.wantOK(t, ok)
			tt.wantErr(t, err)
		})
	}
}
