package processor_test

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/internal/testutil"
	mockNotifier "github.com/clambin/tado-exporter/internal/controller/notifier/mocks"
	"github.com/clambin/tado-exporter/internal/controller/processor"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/action/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestProcessor(t *testing.T) {
	api := mocks.NewTadoSetter(t)

	p := mockPoller.NewPoller(t)
	updateCh := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(updateCh)
	p.EXPECT().Unsubscribe(updateCh).Return()

	s := mockNotifier.NewSlackSender(t)

	f := &fakeEvaluator{}
	l := processor.RulesLoader(func(update poller.Update) (rules.Evaluator, error) {
		return f, nil
	})

	proc := processor.New(api, p, s, l, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		errCh <- proc.Run(ctx)
	}()

	playbook := []struct {
		action  action.Action
		message []slack.Attachment
		task    string
		taskOK  assert.BoolAssertionFunc
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
			task:   "foo: overlay in 1h0m0s",
			taskOK: assert.True,
		},
		{
			action: action.Action{
				Delay:  time.Hour,
				Reason: "reason",
				Label:  "foo",
				State:  testutil.FakeState{ModeValue: action.ZoneInOverlayMode},
			},
			task:   "foo: overlay in 1h0m0s",
			taskOK: assert.True,
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
			taskOK: assert.False,
		},
	}

	for _, entry := range playbook {
		if entry.message != nil {
			s.EXPECT().Send(mock.Anything, entry.message).Return(nil).Once()
		}
		f.set(entry.action)

		updateCh <- poller.Update{}
		// TODO: race condition?
		time.Sleep(100 * time.Millisecond)

		task, ok := proc.ReportTask()
		assert.Equal(t, entry.task, task)
		entry.taskOK(t, ok)
	}

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
