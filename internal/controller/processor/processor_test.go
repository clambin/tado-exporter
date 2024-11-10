package processor_test

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/processor"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/testutil"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestProcessor(t *testing.T) {
	p := mockPoller.NewPoller(t)
	updateCh := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(updateCh)
	p.EXPECT().Unsubscribe(updateCh).Return()

	f := &fakeEvaluator{}
	l := processor.RulesLoader(func(update poller.Update) (rules.Evaluator, error) {
		return f, nil
	})

	proc := processor.New(nil, p, nil, l, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		errCh <- proc.Run(ctx)
	}()

	playbook := []struct {
		action   action.Action
		message  []slack.Attachment
		wantTask string
	}{
		{
			action: action.Action{
				Delay:  time.Hour,
				Reason: "reason",
				Label:  "foo",
				State:  testutil.FakeState{ModeValue: action.ZoneInOverlayMode},
			},
			message: []slack.Attachment{{
				Color: "good",
				Title: "foo: overlay in 1h0m0s",
				Text:  "reason",
			}},
			wantTask: "foo: overlay in 1h0m0s",
		},
		{
			action: action.Action{
				Delay:  time.Hour,
				Reason: "reason",
				Label:  "foo",
				State:  testutil.FakeState{ModeValue: action.ZoneInOverlayMode},
			},
			wantTask: "foo: overlay in 1h0m0s",
		},
		{
			action: action.Action{
				Reason: "reason gone",
				Label:  "foo",
				State:  testutil.FakeState{ModeValue: action.NoAction},
			},
			message: []slack.Attachment{{
				Color: "good",
				Title: "foo: canceling overlay",
				Text:  "reason gone",
			}},
		},
	}

	for _, entry := range playbook {
		f.set(entry.action)

		updateCh <- poller.Update{}

		if entry.wantTask != "" {
			assert.Eventually(t, func() bool {
				task, ok := proc.ReportTask()
				return ok && task == entry.wantTask
			}, time.Second, time.Millisecond)
		}
	}

	cancel()
	assert.NoError(t, <-errCh)

}

func TestScheduler_processResult(t *testing.T) {
	p := mockPoller.NewPoller(t)
	updateCh := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(updateCh)
	p.EXPECT().Unsubscribe(updateCh).Return()
	f := &fakeEvaluator{
		next: action.Action{
			Delay:  100 * time.Millisecond,
			Reason: "test",
			Label:  "test",
			State:  testutil.FakeState{ModeValue: action.ZoneInOverlayMode},
		},
	}
	l := processor.RulesLoader(func(update poller.Update) (rules.Evaluator, error) {
		return f, nil
	})

	proc := processor.New(nil, p, nil, l, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		errCh <- proc.Run(ctx)
	}()

	updateCh <- poller.Update{}
	assert.Eventually(t, func() bool {
		s, ok := proc.ReportTask()
		return ok && s != ""
	}, time.Second, time.Millisecond)

	assert.Eventually(t, func() bool {
		_, ok := proc.ReportTask()
		return !ok
	}, time.Second, 100*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}

var _ rules.Evaluator = &fakeEvaluator{}

type fakeEvaluator struct {
	lock sync.RWMutex
	next action.Action
}

func (f *fakeEvaluator) set(a action.Action) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.next = a
}

func (f *fakeEvaluator) Evaluate(_ poller.Update) (action.Action, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.next, nil
}
