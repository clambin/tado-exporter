package tmp

import (
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"testing"
	"time"
)

var (
	discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
)

/*
	func TestGroupController_zoneRules(t *testing.T) {
		ruleConfig := []RuleConfiguration{
			{
				Name:   "autoAway",
				Script: ScriptConfig{Packaged: `autoaway.lua`},
				Users:  []string{"user A"},
				Args:   Args{"foo": "bar"},
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

		rule, ok := zoneRules[0].(zoneRule)
		require.True(t, ok)
		assert.Equal(t, "bar", rule.args["foo"])

		l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
		tadoClient := mocks.NewTadoClient(t)
		p := fakePublisher{}

		errCh := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())
		g := newGroupController(zoneRules, getZoneStateFromUpdate("zone"), &p, tadoClient, nil, l)
		go func() { errCh <- g.Run(ctx) }()

		require.Eventually(t, func() bool {
			return p.subacribed.Load()
		}, 1*time.Second, 10*time.Millisecond)

		// trigger a deferred action
		p.ch <- testutils.Update(
			testutils.WithHome(1, "my home", tado.HOME),
			testutils.WithZone(1, "zone", tado.PowerON, 21, 18),
			testutils.WithMobileDevice(1, "user A", testutils.WithLocation(false, true)),
		)

		var j *job
		require.Eventually(t, func() bool {
			j = g.scheduledJob.Load()
			return j != nil
		}, time.Second, 10*time.Millisecond)

		assert.Equal(t, 15*time.Minute, j.GetDelay().Round(time.Minute))

		// trigger an immediate action
		tadoClient.EXPECT().
			DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(1)).
			Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
			Once()
		p.ch <- testutils.Update(
			testutils.WithHome(1, "my home", tado.HOME),
			testutils.WithZone(1, "zone", tado.PowerON, 21, 18, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			testutils.WithMobileDevice(1, "user A", testutils.WithLocation(false, true)),
		)

		require.Eventually(t, func() bool {
			return g.scheduledJob.Load() == nil
		}, time.Second, 10*time.Millisecond)

		cancel()
		assert.NoError(t, <-errCh)
	}

var _ Publisher[poller.Update] = &fakePublisher{}

	type fakePublisher struct {
		ch         chan poller.Update
		subacribed atomic.Bool
	}

	func (f *fakePublisher) Subscribe() chan poller.Update {
		f.ch = make(chan poller.Update)
		f.subacribed.Store(true)
		return f.ch
	}

	func (f *fakePublisher) Unsubscribe(_ chan poller.Update) {
		f.subacribed.Store(false)
		f.ch = nil
	}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
*/
func TestGroupController_HomeRules(t *testing.T) {
	// TODO: most of this is already testing at HomeRule level. simplify.
	tests := []struct {
		name string
		update
		want       action
		err        assert.ErrorAssertionFunc
		wantChange assert.BoolAssertionFunc
	}{
		{
			name:       "at least one user home",
			update:     update{nil, devices{{Name: "user A", Home: true}, {Name: "user B", Home: false}}, 1, homeState{false, false}},
			want:       &homeAction{coreAction: coreAction{homeState{true, true}, "one or more users are home: user A", 0}, homeId: 1},
			err:        assert.NoError,
			wantChange: assert.True,
		},
		{
			name:       "all users are away",
			update:     update{nil, devices{{Name: "user A", Home: false}, {Name: "user B", Home: false}}, 1, homeState{true, false}},
			want:       &homeAction{coreAction: coreAction{homeState{false, true}, "all users are away: user A", 5 * time.Minute}, homeId: 1},
			err:        assert.NoError,
			wantChange: assert.True,
		},
		{
			name:       "no devices",
			update:     update{nil, devices{}, 1, homeState{true, false}},
			want:       &homeAction{coreAction: coreAction{homeState{true, false}, "no devices found", 0}, homeId: 1},
			err:        assert.NoError,
			wantChange: assert.False,
		},
		{
			name:       "no change",
			update:     update{nil, devices{{Name: "user A", Home: false}, {Name: "user B", Home: false}}, 1, homeState{false, true}},
			want:       &homeAction{coreAction: coreAction{homeState{false, true}, "all users are away: user A", 0}, homeId: 1},
			err:        assert.NoError,
			wantChange: assert.False,
		},
	}

	rules, err := loadHomeRules([]RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"user A"}},
	})
	assert.NoError(t, err)
	e := newGroupEvaluator(rules, getHomeStateFromUpdate, nil, nil, nil, discardLogger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			a, change, err := e.evaluate(tt.update)
			tt.err(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, a)
			tt.wantChange(t, change)
		})
	}
}

/*
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
		rules []tmp.evaluator
		tmp.update
		want
	}{
		{
			name: "no rules",
			want: want{"", 0, "no rules found", assert.Error},
		},
		{
			name: "single rule",
			rules: []tmp.evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
			},
			update: tmp.update{homeState: HomeStateAuto, ZoneStates: map[string]tmp.zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 0, "no manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with same desired zone state: pick the first one",
			rules: []tmp.evaluator{
				fakeZoneEvaluator{ZoneStateAuto, time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateAuto, time.Hour, "manual setting detected", nil},
			},
			update: tmp.update{homeState: HomeStateAuto, ZoneStates: map[string]tmp.zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices: nil},
			want:   want{ZoneStateAuto, time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states: pick the first one",
			rules: []tmp.evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
			},
			update: tmp.update{homeState: HomeStateAuto, ZoneStates: map[string]tmp.zoneInfo{"foo": {zoneState: ZoneStateManual}}, devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different desired zone states, including `no change`: pick the first non-matching",
			rules: []tmp.evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 5 * time.Minute, "manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateOff, time.Hour, "no users home", nil},
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
			},
			update: tmp.update{homeState: HomeStateAuto, ZoneStates: map[string]tmp.zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
			want:   want{ZoneStateAuto, 5 * time.Minute, "manual setting detected", assert.NoError},
		},
		{
			name: "multiple rules with different 'no-change' actions: join the reasons",
			rules: []tmp.evaluator{
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
				fakeZoneEvaluator{ZoneStateAuto, 0, "users are home", nil},
				fakeZoneEvaluator{ZoneStateAuto, 0, "no manual setting detected", nil},
			},
			update: tmp.update{homeState: HomeStateAuto, ZoneStates: map[string]tmp.zoneInfo{"foo": {zoneState: ZoneStateAuto}}, devices: nil},
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
	autoAwayCfg := tmp.RuleConfiguration{"", tmp.ScriptConfig{Packaged: "autoaway.lua"}, []string{"user"}, nil}
	limitOverlayCfg := tmp.RuleConfiguration{"", tmp.ScriptConfig{Packaged: "limitoverlay.lua"}, nil, nil}

	tests := []struct {
		name   string
		rules  []tmp.RuleConfiguration
		update tmp.update
		want   action
	}{
		{
			name:  "user is home: no action",
			rules: []tmp.RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: tmp.update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]tmp.zoneInfo{"zone": {zoneState: ZoneStateAuto, ZoneId: 1}},
				devices:    []device{{"user", true}},
			},
			want: &zoneAction{ZoneStateAuto, 0, "no manual setting detected, one or more users are home: user", 1, 1, "zone"},
		},
		{
			name:  "user is not home: switch heating is off",
			rules: []tmp.RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: tmp.update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]tmp.zoneInfo{"zone": {ZoneStateAuto, 1}},
				devices:    []device{{"user", false}},
			},
			want: &zoneAction{ZoneStateOff, 15 * time.Minute, "all users are away", 1, 1, "zone"},
		},
		{
			name:  "user is not home, heating is off: no action",
			rules: []tmp.RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: tmp.update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]tmp.zoneInfo{"zone": {ZoneStateOff, 1}},
				devices:    []device{{"user", false}},
			},
			want: &zoneAction{ZoneStateOff, 15 * time.Minute, "all users are away", 1, 1, "zone"},
		},
		{
			name:  "user is home, heating is off: move heating to auto mode",
			rules: []tmp.RuleConfiguration{limitOverlayCfg, autoAwayCfg},
			update: tmp.update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]tmp.zoneInfo{"zone": {ZoneStateOff, 1}},
				devices:    []device{{"user", true}},
			},
			want: &zoneAction{ZoneStateAuto, 0, "one or more users are home: user", 1, 1, "zone"},
		},
		{
			name:  "user is home, zone in manual mode: schedule auto mode",
			rules: []tmp.RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: tmp.update{
				homeState:  HomeStateHome,
				HomeId:     1,
				ZoneStates: map[string]tmp.zoneInfo{"zone": {zoneState: ZoneStateManual, ZoneId: 1}},
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
*/
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func Test_shouldSchedule(t *testing.T) {
	tests := []struct {
		name    string
		action  action
		job     scheduledJob
		isNewer assert.BoolAssertionFunc
	}{
		{
			name:    "action is different: schedule",
			action:  &homeAction{coreAction{homeState{true, false}, "", time.Hour}, 1},
			job:     fakeScheduledJob{state: homeState{false, true}, due: time.Now()},
			isNewer: assert.True,
		},
		/*
			{
				name:    "action is earlier: schedule",
				action:  &zoneAction{zoneState: ZoneStateAuto, delay: 0},
				job:     fakeScheduledJob{state: string(ZoneStateAuto), due: time.Now().Add(time.Hour)},
				isNewer: assert.True,
			},
			{
				name:    "action is later: don't schedule",
				action:  &zoneAction{zoneState: ZoneStateAuto, delay: time.Hour},
				job:     fakeScheduledJob{state: string(ZoneStateAuto), due: time.Now()},
				isNewer: assert.False,
			},
			{
				name:   "due date is rounded to minutes",
				action: &zoneAction{zoneState: ZoneStateAuto, delay: 15 * time.Second},
				job:    fakeScheduledJob{state: string(ZoneStateAuto), due: time.Now()},

				isNewer: assert.False,
			},*/
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.isNewer(t, shouldSchedule(tt.job, tt.action))
		})
	}
}

var _ scheduledJob = fakeScheduledJob{}

type fakeScheduledJob struct {
	state state
	due   time.Time
}

func (f fakeScheduledJob) GetState() state {
	return f.state
}

func (f fakeScheduledJob) Due() time.Time {
	return f.due
}
