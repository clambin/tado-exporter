package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroupEvaluator_ScheduleAndCancel(t *testing.T) {
	rules, err := loadZoneRules(
		"zone",
		[]RuleConfiguration{
			{Name: "limitOverlay", Script: ScriptConfig{Packaged: "limitoverlay.lua"}},
		},
	)
	require.NoError(t, err)

	var p fakePublisher
	n := fakeNotifier{ch: make(chan string)}
	e := newGroupEvaluator(rules, getZoneStateFromUpdate("zone"), &p, nil, &n, discardLogger)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- e.Run(ctx) }()

	// wait for the group evaluator to subscribe to the publishes
	assert.Eventually(t, func() bool { return p.subscribed.Load() }, time.Second, time.Millisecond)

	// zone is in overlay
	go func() {
		p.ch <- testutils.Update(
			testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
		)
	}()

	const want = "*zone*: switching heating to auto mode in 1h0m0s\nReason: manual setting detected"
	assert.Equal(t, want, <-n.ch)
	assert.Equal(t, want, e.ReportTask())

	go func() {
		// same update: don't schedule a new job
		p.ch <- testutils.Update(
			testutils.WithZone(10, "zone", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
		)
		// zone is back in auto mode
		p.ch <- testutils.Update(
			testutils.WithZone(10, "zone", tado.PowerON, 21, 20),
		)
	}()
	assert.Equal(t, "*zone*: switching heating to auto mode canceled\nReason: no manual setting detected", <-n.ch)
	assert.Eventually(t, func() bool { return e.ReportTask() == "" }, time.Second, time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}

func TestGroupEvaluator_Do(t *testing.T) {
	rules, err := loadZoneRules(
		"zone",
		[]RuleConfiguration{
			{Name: "autoAway", Script: ScriptConfig{Packaged: "autoaway.lua"}},
		},
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	var p fakePublisher
	tadoClient := mocks.NewTadoClient(t)
	tadoClient.EXPECT().
		DeleteZoneOverlayWithResponse(ctx, tado.HomeId(1), tado.ZoneId(10)).
		Return(&tado.DeleteZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusNoContent}}, nil).
		Once()
	n := fakeNotifier{ch: make(chan string)}
	e := newGroupEvaluator(rules, getZoneStateFromUpdate("zone"), &p, tadoClient, &n, discardLogger)

	errCh := make(chan error)
	go func() { errCh <- e.Run(ctx) }()

	// wait for the group evaluator to subscribe to the publishes
	assert.Eventually(t, func() bool { return p.subscribed.Load() }, time.Second, time.Millisecond)

	// zone is off but user is home: remove overlay immediately
	go func() {
		p.ch <- testutils.Update(
			testutils.WithZone(10, "zone", tado.PowerOFF, 0, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, false)),
		)
	}()
	assert.Equal(t, "*zone*: switching heating to auto mode\nReason: one or more users are home: user", <-n.ch)

	cancel()
	assert.NoError(t, <-errCh)
}

func TestGroupEvaluator(t *testing.T) {
	rules, err := loadHomeRules([]RuleConfiguration{
		{Name: "autoAway", Script: ScriptConfig{Packaged: "homeandaway.lua"}, Users: []string{"user A"}},
	})
	assert.NoError(t, err)

	tadoClient := mocks.NewTadoClient(t)
	p := fakePublisher{}

	e := newGroupEvaluator(rules, getHomeStateFromUpdate, &p, tadoClient, nil, discardLogger)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- e.Run(ctx) }()

	// wait for the group evaluator to subscribe to the publishes
	assert.Eventually(t, func() bool { return p.subscribed.Load() }, time.Second, time.Millisecond)

	// all users away -> schedule moving home to AWAY mode
	p.ch <- testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME, testutils.WithPresenceLocked(true)),
		testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
	)

	// wait for group evaluator to process the update. should result in scheduled job
	require.Eventually(t, func() bool { return e.ReportTask() != "" }, time.Second, time.Millisecond)
	assert.Equal(t, "setting home to AWAY mode in 5m0s\nReason: all users are away: user A", e.ReportTask())

	// user comes home -> action should be canceled
	p.ch <- testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME, testutils.WithPresenceLocked(true)),
		testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
	)
	require.Eventually(t, func() bool { return e.ReportTask() == "" }, time.Second, time.Millisecond)

	// user comes home, home is in manual away mode -> set home to home mode
	var done atomic.Bool
	tadoClient.EXPECT().
		SetPresenceLockWithResponse(ctx, tado.HomeId(1), mock.AnythingOfType("tado.PresenceLock")).
		RunAndReturn(func(_ context.Context, _ int64, lock tado.PresenceLock, _ ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error) {
			assert.Equal(t, tado.HOME, *lock.HomePresence)
			done.Store(true)
			return &tado.SetPresenceLockResponse{HTTPResponse: &http.Response{StatusCode: http.StatusOK}}, nil
		}).
		Once()
	p.ch <- testutils.Update(
		testutils.WithHome(1, "my home", tado.AWAY, testutils.WithPresenceLocked(true)),
		testutils.WithMobileDevice(100, "user A", testutils.WithLocation(true, false)),
	)

	assert.Eventually(t, func() bool { return done.Load() }, time.Second, time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func TestGroupController_ZoneRules_AutoAway_vs_LimitOverlay(t *testing.T) {
	autoAwayCfg := RuleConfiguration{
		Name:   "autoAway",
		Script: ScriptConfig{Packaged: "autoaway.lua"},
		Users:  []string{"user"},
	}
	limitOverlayCfg := RuleConfiguration{
		Name:   "limitOverlay",
		Script: ScriptConfig{Packaged: "limitoverlay.lua"},
	}

	tests := []struct {
		name     string
		rules    []RuleConfiguration
		update   poller.Update
		isChange assert.BoolAssertionFunc
		want     action
	}{
		{
			name:  "user is home: no action",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 21),
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, false)),
			),
			isChange: assert.False,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{false, true},
					reason: "no manual setting detected, one or more users are home: user",
					delay:  0,
				},
				zoneName: "zone",
				homeId:   1,
				zoneId:   10,
			},
		},
		{
			name:  "user is not home, heating is on: switch heating is off",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 21),
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(false, false)),
			),
			isChange: assert.True,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{true, false},
					delay:  15 * time.Minute,
					reason: "all users are away: user",
				},
				zoneName: "zone",
				homeId:   1,
				zoneId:   10,
			},
		},
		{
			name:  "user is not home, heating is off: no action",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 21, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(false, false)),
			),
			isChange: assert.False,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{true, false},
					delay:  0,
					reason: "all users are away: user, heating is off",
				},
				zoneName: "zone",
				homeId:   1,
				zoneId:   10,
			},
		},
		{
			name:  "user is home, heating is off: move heating to auto mode",
			rules: []RuleConfiguration{limitOverlayCfg, autoAwayCfg},
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerOFF, 21, 21, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithMobileDevice(100, "user", testutils.WithLocation(true, false)),
			),
			isChange: assert.True,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{false, true},
					delay:  0,
					reason: "one or more users are home: user",
				},
				zoneName: "zone",
				homeId:   1,
				zoneId:   10,
			},
		},
		{
			name:  "user is home, zone in manual mode: schedule auto mode",
			rules: []RuleConfiguration{autoAwayCfg, limitOverlayCfg},
			update: testutils.Update(
				testutils.WithZone(10, "zone", tado.PowerON, 21, 21, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			isChange: assert.True,
			want: &zoneAction{
				coreAction: coreAction{
					state:  zoneState{false, true},
					delay:  time.Hour,
					reason: "manual setting detected",
				},
				zoneName: "zone",
				homeId:   1,
				zoneId:   10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zr, err := loadZoneRules("zone", tt.rules)
			require.NoError(t, err)
			e := newGroupEvaluator(zr, getZoneStateFromUpdate("zone"), nil, nil, nil, discardLogger)

			a, change, err := e.evaluate(tt.update)
			require.NoError(t, err)
			tt.isChange(t, change)
			assert.Equal(t, tt.want, a)
		})
	}
}

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
		{
			name:    "action is earlier: schedule",
			action:  &homeAction{coreAction{homeState{true, false}, "", 0}, 1},
			job:     fakeScheduledJob{state: homeState{true, false}, due: time.Now().Add(time.Hour)},
			isNewer: assert.True,
		},
		{
			name:    "action is later: don't schedule",
			action:  &homeAction{coreAction{homeState{true, false}, "", time.Hour}, 1},
			job:     fakeScheduledJob{state: homeState{true, false}, due: time.Now()},
			isNewer: assert.False,
		},
		{
			name:    "due time is rounded to minutes",
			action:  &homeAction{coreAction{homeState{true, false}, "", 15 * time.Second}, 1},
			job:     fakeScheduledJob{state: homeState{true, false}, due: time.Now()},
			isNewer: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.isNewer(t, shouldSchedule(tt.job, tt.action))
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Notifier = &fakeNotifier{}

type fakeNotifier struct {
	ch chan string
}

func (f fakeNotifier) Notify(s string) {
	f.ch <- s
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ scheduledJob = fakeScheduledJob{}

type fakeScheduledJob struct {
	state state
	due   time.Time
}

func (f fakeScheduledJob) State() state {
	return f.state
}

func (f fakeScheduledJob) Due() time.Time {
	return f.due
}

var _ Publisher[poller.Update] = &fakePublisher{}

type fakePublisher struct {
	ch         chan poller.Update
	subscribed atomic.Bool
}

func (f *fakePublisher) Subscribe() chan poller.Update {
	f.ch = make(chan poller.Update)
	f.subscribed.Store(true)
	return f.ch
}

func (f *fakePublisher) Unsubscribe(_ chan poller.Update) {
	f.subscribed.Store(false)
	f.ch = nil
}
