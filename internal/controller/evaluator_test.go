package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/mocks"
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
	r, err := rules.LoadZoneRules(
		"zone",
		[]rules.RuleConfiguration{
			{Name: "limitOverlay", Script: rules.ScriptConfig{Packaged: "limitoverlay"}},
		},
	)
	require.NoError(t, err)

	var p fakePublisher
	n := fakeNotifier{ch: make(chan string)}
	e := newGroupEvaluator(r, &p, nil, &n, discardLogger)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- e.Run(ctx) }()

	// wait for the group rule to subscribe to the publishes
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
	r, err := rules.LoadZoneRules(
		"zone",
		[]rules.RuleConfiguration{
			{Name: "autoAway", Script: rules.ScriptConfig{Packaged: "autoaway"}},
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
	e := newGroupEvaluator(r, &p, tadoClient, &n, discardLogger)

	errCh := make(chan error)
	go func() { errCh <- e.Run(ctx) }()

	// wait for the group rule to subscribe to the publishes
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
	r, err := rules.LoadHomeRules([]rules.RuleConfiguration{
		{Name: "autoAway", Script: rules.ScriptConfig{Packaged: "homeandaway"}, Users: []string{"user A"}},
	})
	assert.NoError(t, err)

	tadoClient := mocks.NewTadoClient(t)
	p := fakePublisher{}

	e := newGroupEvaluator(r, &p, tadoClient, nil, discardLogger)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() { errCh <- e.Run(ctx) }()

	// wait for the group rule to subscribe to the publishes
	assert.Eventually(t, func() bool { return p.subscribed.Load() }, time.Second, time.Millisecond)

	// all users away -> schedule moving home to AWAY mode
	p.ch <- testutils.Update(
		testutils.WithHome(1, "my home", tado.HOME, testutils.WithPresenceLocked(true)),
		testutils.WithMobileDevice(100, "user A", testutils.WithLocation(false, false)),
	)

	// wait for group rule to process the update. should result in scheduled job
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

var _ notifier.Notifier = &fakeNotifier{}

type fakeNotifier struct {
	ch chan string
}

func (f fakeNotifier) Notify(s string) {
	f.ch <- s
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Publisher[poller.Update] = &fakePublisher{}

type fakePublisher struct {
	ch         chan poller.Update
	subscribed atomic.Bool
}

func (f *fakePublisher) Subscribe() <-chan poller.Update {
	f.ch = make(chan poller.Update)
	f.subscribed.Store(true)
	return f.ch
}

func (f *fakePublisher) Unsubscribe(_ <-chan poller.Update) {
	f.subscribed.Store(false)
	f.ch = nil
}
