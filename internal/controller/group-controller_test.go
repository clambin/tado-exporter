package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestGroupController(t *testing.T) {
	ruleConfig := []RuleConfiguration{
		{
			Name:   "autoAway",
			Script: ScriptConfig{Packaged: `autoaway.lua`},
			Users:  []string{"user A"},
		},
		{
			Name: "limitOverlay",
			Script: ScriptConfig{Text: `function Evaluate(_, zone, _)
	if zone == "auto" then
		return "auto", 0, "no manual setting detected"
	end
	return "auto", 0, "manual setting detected"
end
`},
		},
	}
	zoneRules, err := loadZoneRules("zone", ruleConfig)
	require.NoError(t, err)
	require.Len(t, zoneRules, 2)

	ctx := context.Background()
	//f := fakeNotifier{ch: make(chan string)}
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tadoClient := mocks.NewTadoClient(t)
	g := newGroupController(zoneRules, getZoneStateFromUpdate("zone"), nil, tadoClient, nil, l)

	u := testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME),
		testutils.WithZone(1, "zone", tado.PowerON, 21, 18),
		testutils.WithMobileDevice(1, "user A", testutils.WithLocation(false, true)),
	)

	a, ok := g.processUpdate(u)
	assert.True(t, ok)
	assert.Equal(t, 15*time.Minute, a.GetDelay())

	g.scheduleJob(ctx, a)
	j := g.scheduledJob.Load()
	require.NotNil(t, j)
	assert.Equal(t, 15*time.Minute, j.GetDelay().Round(time.Minute))

	u = testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME),
		testutils.WithZone(1, "zone", tado.PowerON, 21, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
		testutils.WithMobileDevice(1, "user A", testutils.WithLocation(false, true)),
	)

	a, ok = g.processUpdate(u)
	assert.True(t, ok)
	assert.Equal(t, time.Duration(0), a.GetDelay())

	tadoClient.EXPECT().
		DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(1)).
		Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
		Once()
	g.scheduleJob(ctx, a)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestGroupController_HomeRules(t *testing.T) {
	tests := []struct {
		name string
		update
		homeWant
	}{
		{
			name: "at least one user home",
			update: update{HomeStateAway, 1, nil, devices{
				{Name: "user A", Home: true},
				{Name: "user B", Home: false},
			}},
			homeWant: homeWant{HomeStateHome, 0, "one or more users are home: user A", assert.NoError},
		},
		{
			name: "all users are away",
			update: update{HomeStateHome, 1, nil, devices{
				{Name: "user A", Home: false},
				{Name: "user B", Home: false},
			}},
			homeWant: homeWant{HomeStateAway, 5 * time.Minute, "all users are away", assert.NoError},
		},
		{
			name:     "no devices",
			update:   update{HomeStateAway, 1, nil, devices{}},
			homeWant: homeWant{HomeStateAway, 0, "no devices found", assert.NoError},
		},
	}

	rules, err := loadHomeRules([]RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"user A"}},
	})
	assert.NoError(t, err)
	e := newGroupController(rules, getHomeStateFromUpdate, nil, nil, nil, discardLogger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			a, change, err := e.evaluate(tt.update, &homeAction{state: HomeStateAuto})
			tt.homeWant.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.homeWant.state, homeState(a.GetState()))
			assert.Equal(t, tt.homeWant.delay, a.GetDelay())
			assert.Equal(t, tt.homeWant.reason, a.GetReason())
			if a.GetState() != string(HomeStateAuto) {
				assert.True(t, change)
			} else {
				assert.False(t, change)
			}
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestGroupController_ZoneRules(t *testing.T) {
	type want struct {
		zoneState
		delay  time.Duration
		reason string
		err    assert.ErrorAssertionFunc
	}
	tests := []struct {
		name  string
		rules []evaluator
		update
		want
	}{
		{
			name: "no rules",
			want: want{"", 0, "no rules found", assert.Error},
		},
		{
			name: "single rule",
			rules: []evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with same desired zone state: pick the first one",
			rules: []evaluator{
				fakeZoneEvaluator{ZoneStateAuto, time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateAuto, time.Hour, "manual setting detected", nil},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices: nil},
			want:   want{ZoneStateAuto, time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states: pick the first one",
			rules: []evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states, including `no change`: pick the first non-matching",
			rules: []evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different 'no-change' actions: join the reasons",
			rules: []evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateAuto, 0, "users are home", nil},
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
			},
			update: update{homeState: HomeStateAuto, ZoneStates: map[string]zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 0, "no manual setting detected, users are home", assert.NoError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newGroupController(tt.rules, getZoneStateFromUpdate("foo"), nil, nil, nil, discardLogger)
			a, _, err := e.evaluate(tt.update, &zoneAction{zoneState: ZoneStateAuto})
			tt.want.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want.zoneState, zoneState(a.GetState()))
			assert.Equal(t, tt.want.delay, a.GetDelay())
			assert.Equal(t, tt.want.reason, a.GetReason())
		})
	}
}

func TestGroupController_ZoneRules_AutoAway_vs_LimitOverlay(t *testing.T) {
	autoAwayCfg := RuleConfiguration{"", ScriptConfig{Packaged: "autoaway.lua"}, []string{"user"}}
	limitOverlayCfg := RuleConfiguration{"", ScriptConfig{Packaged: "limitoverlay.lua"}, nil}

	tests := []struct {
		name   string
		rules  []RuleConfiguration
		update update
		want   action
	}{
		{
			name:  "user is home: no action",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {zoneState: ZoneStateAuto, ZoneId: 1}},
				devices:    []device{{"user", true}},
			},
			want: &zoneAction{ZoneStateAuto, 0, "no manual setting detected, one or more users are home: user", 1, 1, "zone"},
		},
		{
			name:  "user is not home: switch heating is off",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {ZoneStateAuto, 1}},
				devices:    []device{{"user", false}},
			},
			want: &zoneAction{ZoneStateOff, 15 * time.Minute, "all users are away", 1, 1, "zone"},
		},
		{
			name:  "user is not home, heating is off: no action",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {ZoneStateOff, 1}},
				devices:    []device{{"user", false}},
			},
			want: &zoneAction{ZoneStateOff, 15 * time.Minute, "all users are away", 1, 1, "zone"},
		},
		{
			name:  "user is home, heating is off: move heating to auto mode",
			rules: []RuleConfiguration{limitOverlayCfg, autoAwayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {ZoneStateOff, 1}},
				devices:    []device{{"user", true}},
			},
			want: &zoneAction{ZoneStateAuto, 0, "one or more users are home: user", 1, 1, "zone"},
		},
		{
			name:  "user is home, zone in manual mode: schedule auto mode",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]zoneInfo{"zone": {zoneState: ZoneStateManual, ZoneId: 1}},
				devices:    []device{{"user", true}},
			},
			want: &zoneAction{ZoneStateAuto, time.Hour, "manual setting detected", 1, 1, "zone"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zr, err := loadZoneRules("zone", tt.rules)
			require.NoError(t, err)
			e := newGroupController(zr, getZoneStateFromUpdate("zone"), nil, nil, nil, discardLogger)

			current, ok := tt.update.GetZoneState("zone")
			require.True(t, ok)
			a, _, err := e.evaluate(tt.update, &zoneAction{zoneState: current})
			require.NoError(t, err)
			assert.Equal(t, tt.want, a)
		})
	}
}
