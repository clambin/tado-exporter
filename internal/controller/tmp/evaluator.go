package tmp

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

// An evaluator takes the current update and determines the next action, based on one or more rules.
type evaluator interface {
	Evaluate(update) (action, error)
}

// A groupEvaluator evaluates all rules for a given home or zone. It receives updates from a Poller, evaluates all rules
// and executes the required action. If the required action has a configured delay, it schedules a job and manages its lifetime.
//
// groupEvaluator uses a Notifier (which may be a list of Notifiers) to inform the user.
type groupEvaluator struct {
	Publisher[poller.Update]
	TadoClient
	Notifier
	getState     func(update) (state, error)
	logger       *slog.Logger
	jobCompleted chan struct{}
	scheduledJob atomic.Pointer[job]
	rules        []evaluator
}

// newGroupEvaluator creates a new controller for the provided rules.
func newGroupEvaluator(
	rules []evaluator,
	getState func(update) (state, error),
	p Publisher[poller.Update],
	client TadoClient,
	notifier Notifier,
	l *slog.Logger,
) *groupEvaluator {
	return &groupEvaluator{
		Publisher:    p,
		TadoClient:   client,
		Notifier:     notifier,
		logger:       l,
		jobCompleted: make(chan struct{}),

		rules:    rules,
		getState: getState,
	}
}

// Run registers with a Poller and evaluates an incoming update against its rules.
func (c *groupEvaluator) Run(ctx context.Context) error {
	ch := c.Publisher.Subscribe()
	defer c.Publisher.Unsubscribe(ch)

	c.logger.Debug("group controller starting")
	defer c.logger.Debug("group controller stopping")

	for {
		select {
		case <-ctx.Done():
			return nil
		case u := <-ch:
			if a, ok := c.processUpdate(u); ok {
				c.scheduleJob(ctx, a)
			} else {
				c.cancelJob(a)
			}
		case <-c.jobCompleted:
			c.processCompletedJob()
		}
	}
}

// processUpdate processes the update, evaluating its rules. If the outcome differs from the current state
// (as determined by the update), it returns the action and true. Otherwise, it returns false.
func (c *groupEvaluator) processUpdate(update poller.Update) (action, bool) {
	u := updateFromPollerUpdate(update)
	a, change, err := c.evaluate(u)
	if err != nil {
		c.logger.Error("failed to evaluate zone rules", "err", err)
		return nil, false
	}
	return a, change
}

func (c *groupEvaluator) evaluate(u update) (action, bool, error) {
	if len(c.rules) == 0 {
		return nil, false, errors.New("no rules found")
	}

	current, err := c.getState(u)
	if err != nil {
		c.logger.Error("failed to parse update", "err", err)
		return nil, false, fmt.Errorf("failed to determine current state: %w", err)
	}

	noChange := make([]action, 0, len(c.rules))
	change := make([]action, 0, len(c.rules))
	for i := range c.rules {
		a, err := c.rules[i].Evaluate(u)
		if err != nil {
			return nil, false, fmt.Errorf("failed to evaluate rule %d: %w", i+1, err)
		}
		if a.GetState().Equals(current) && a.GetDelay() == 0 {
			noChange = append(noChange, a)
		} else {
			change = append(change, a)
		}
	}
	if len(change) > 0 {
		slices.SortFunc(change, func(a, b action) int {
			return cmp.Compare(a.GetDelay(), b.GetDelay())
		})
		return change[0], true, nil
	}
	reasons := set.New[string]()
	for _, a := range noChange {
		reasons.Add(a.GetReason())
	}
	noChange[0].setReason(strings.Join(reasons.ListOrdered(), ", "))

	return noChange[0], false, nil
}

// scheduleJob is called when processUpdate returns a new action. It executes (or schedules) the required action.
func (c *groupEvaluator) scheduleJob(ctx context.Context, action action) {
	// if a job is scheduled with the same action, but an earlier scheduled time, don't schedule a new job
	j := c.scheduledJob.Load()
	if j != nil {
		if !shouldSchedule(j, action) {
			return
		}
		// scheduling a new job. cancel any old one.
		c.cancelJob(action)
	}

	// immediate action
	if action.GetDelay() == 0 {
		_ = c.doAction(ctx, action)
		return
	}

	// deferred action
	j = &job{
		TadoClient: c.TadoClient,
		action:     action,
		Job: scheduler.Schedule(ctx, scheduler.RunFunc(func(ctx context.Context) error {
			return c.doAction(ctx, action)
		}), action.GetDelay(), c.jobCompleted),
	}
	c.scheduledJob.Store(j)
	if c.Notifier != nil {
		c.Notifier.Notify(action.Description(true) + "\nReason: " + action.GetReason())
	}
}

// shouldSchedule returns true if the newAction should be scheduled, i.e. either the action is different than the scheduled action,
// or newAction should run before the scheduled action.
func shouldSchedule(currentJob scheduledJob, newAction action) bool {
	if !currentJob.GetState().Equals(newAction.GetState()) {
		return true
	}
	// truncate old & new due times up to a minute and only start a new job (canceling the old one) if newDue is after due.
	// this avoids canceling the current job & immediately scheduling a new one if the old & new due times are very close
	// (e.g. in case of a rule like nighttime, which targets a specific time of day.
	due := currentJob.Due().Truncate(time.Minute)
	newDue := time.Now().Local().Add(newAction.GetDelay()).Truncate(time.Minute)
	return newDue.Before(due)
}

// doAction executes the action and reports the result to the user through a Notifier.
// This is called either directly from scheduleJob, or from the scheduler once the Delay has passed.
func (c *groupEvaluator) doAction(ctx context.Context, action action) error {
	if err := action.Do(ctx, c.TadoClient); err != nil {
		c.logger.Error("failed to execute action", "action", action, "err", err)
		return err
	}
	if c.Notifier != nil {
		c.Notifier.Notify(action.Description(false) + "\nReason: " + action.GetReason())
	}
	return nil
}

// cancelJob cancels any scheduled job.
func (c *groupEvaluator) cancelJob(a action) {
	if j := c.scheduledJob.Load(); j != nil {
		j.Cancel()
		if c.Notifier != nil {
			c.Notifier.Notify(j.Description(true) + " canceled\nReason: " + a.GetReason())
		}
	}
}

// processCompletedJob is notified by the scheduler once the job has completed and informs the user through a Notifier.
func (c *groupEvaluator) processCompletedJob() {
	if j := c.scheduledJob.Load(); j != nil {
		defer c.scheduledJob.Store(nil)
		_, err := j.Result()
		if err != nil && !errors.Is(err, context.Canceled) {
			c.logger.Error("scheduled job failed", "err", err)
			return
		}
	}
}

func (c *groupEvaluator) ReportTask() string {
	if j := c.scheduledJob.Load(); j != nil {
		return j.Description(false) + " in " + time.Until(j.Due()).Truncate(time.Second).String()
	}
	return ""
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
var _ scheduler.Runnable = &job{}

var _ scheduledJob = &job{}

type scheduledJob interface {
	Due() time.Time
	GetState() state
}

type job struct {
	action
	TadoClient
	*scheduler.Job
}

func (j job) Run(_ context.Context) error {
	return j.action.Do(context.Background(), j.TadoClient)
}
