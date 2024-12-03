package controller

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
	Evaluate(poller.Update) (action, error)
}

// A groupEvaluator evaluates all rules for a given home or zone. It receives updates from a Poller, evaluates all rules
// and executes the required action. If the required action has a configured delay, it schedules a job and manages its lifetime.
//
// groupEvaluator uses a Notifier (which may be a list of Notifiers) to inform the user.
type groupEvaluator struct {
	Publisher[poller.Update]
	TadoClient
	Notifier
	getState     func(poller.Update) (state, error)
	logger       *slog.Logger
	jobCompleted chan struct{}
	scheduledJob atomic.Pointer[job]
	rules        []evaluator
}

// newGroupEvaluator creates a new controller for the provided rules.
func newGroupEvaluator(
	rules []evaluator,
	getState func(poller.Update) (state, error),
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
func (g *groupEvaluator) Run(ctx context.Context) error {
	ch := g.Publisher.Subscribe()
	defer g.Publisher.Unsubscribe(ch)

	g.logger.Debug("group controller starting")
	defer g.logger.Debug("group controller stopping")

	for {
		select {
		case <-ctx.Done():
			return nil
		case u := <-ch:
			if a, ok := g.processUpdate(u); ok {
				g.scheduleJob(ctx, a)
			} else {
				g.cancelJob(a)
			}
		case <-g.jobCompleted:
			g.processCompletedJob()
		}
	}
}

// processUpdate processes the update, evaluating its rules. If the outcome differs from the current state
// (as determined by the update), it returns the action and true. Otherwise, it returns false.
func (g *groupEvaluator) processUpdate(update poller.Update) (action, bool) {
	a, change, err := g.evaluate(update)
	if err != nil {
		g.logger.Error("failed to evaluate zone rules", "err", err)
		return nil, false
	}
	return a, change
}

func (g *groupEvaluator) evaluate(update poller.Update) (action, bool, error) {
	if len(g.rules) == 0 {
		return nil, false, errors.New("no rules found")
	}

	current, err := g.getState(update)
	if err != nil {
		g.logger.Error("failed to parse update", "err", err)
		return nil, false, fmt.Errorf("failed to determine current state: %w", err)
	}

	noChange := make([]action, 0, len(g.rules))
	change := make([]action, 0, len(g.rules))
	for i := range g.rules {
		a, err := g.rules[i].Evaluate(update)
		if err != nil {
			return nil, false, fmt.Errorf("failed to evaluate rule %d: %w", i+1, err)
		}
		if a.State().Equals(current) && a.Delay() == 0 {
			noChange = append(noChange, a)
		} else {
			change = append(change, a)
		}
	}
	if len(change) > 0 {
		slices.SortFunc(change, func(a, b action) int {
			return cmp.Compare(a.Delay(), b.Delay())
		})
		return change[0], true, nil
	}
	reasons := set.New[string]()
	for _, a := range noChange {
		reasons.Add(a.Reason())
	}
	noChange[0].setReason(strings.Join(reasons.ListOrdered(), ", "))

	return noChange[0], false, nil
}

// scheduleJob is called when processUpdate returns a new action. It executes (or schedules) the required action.
func (g *groupEvaluator) scheduleJob(ctx context.Context, action action) {
	// if a job is scheduled with the same action, but an earlier scheduled time, don't schedule a new job
	j := g.scheduledJob.Load()
	if j != nil {
		if !shouldSchedule(j, action) {
			return
		}
		// scheduling a new job. cancel any old one.
		g.cancelJob(action)
	}

	// immediate action
	if action.Delay() == 0 {
		_ = g.doAction(ctx, action)
		return
	}

	// deferred action
	j = &job{
		TadoClient: g.TadoClient,
		action:     action,
		Job: scheduler.Schedule(ctx, scheduler.RunFunc(func(ctx context.Context) error {
			return g.doAction(ctx, action)
		}), action.Delay(), g.jobCompleted),
		logger: g.logger,
	}
	g.scheduledJob.Store(j)
	if g.Notifier != nil {
		g.Notifier.Notify(action.Description(true) + "\nReason: " + action.Reason())
	}
}

// shouldSchedule returns true if the newAction should be scheduled, i.e. either the action is different than the scheduled action,
// or newAction should run before the scheduled action.
func shouldSchedule(currentJob scheduledJob, newAction action) bool {
	if !currentJob.State().Equals(newAction.State()) {
		return true
	}
	// truncate old & new due times up to a minute and only start a new job (canceling the old one) if newDue is after due.
	// this avoids canceling the current job & immediately scheduling a new one if the old & new due times are very close
	// (e.g. in case of a rule like nighttime, which targets a specific time of day.
	due := currentJob.Due().Truncate(time.Minute)
	newDue := time.Now().Local().Add(newAction.Delay()).Truncate(time.Minute)
	return newDue.Before(due)
}

// doAction executes the action and reports the result to the user through a Notifier.
// This is called either directly from scheduleJob, or from the scheduler once the Delay has passed.
func (g *groupEvaluator) doAction(ctx context.Context, action action) error {
	if err := action.Do(ctx, g.TadoClient, g.logger); err != nil {
		g.logger.Error("failed to execute action", "action", action, "err", err)
		return err
	}
	if g.Notifier != nil {
		g.Notifier.Notify(action.Description(false) + "\nReason: " + action.Reason())
	}
	return nil
}

// cancelJob cancels any scheduled job.
func (g *groupEvaluator) cancelJob(a action) {
	if j := g.scheduledJob.Load(); j != nil {
		j.Cancel()
		if g.Notifier != nil {
			g.Notifier.Notify(j.Description(false) + " canceled\nReason: " + a.Reason())
		}
	}
}

// processCompletedJob is notified by the scheduler once the job has completed and informs the user through a Notifier.
func (g *groupEvaluator) processCompletedJob() {
	if j := g.scheduledJob.Load(); j != nil {
		defer g.scheduledJob.Store(nil)
		if _, err := j.Result(); err != nil && !errors.Is(err, context.Canceled) {
			g.logger.Error("scheduled job failed", "err", err)
		}
	}
}

func (g *groupEvaluator) ReportTask() string {
	if j := g.scheduledJob.Load(); j != nil {
		return j.Description(false) + " in " + time.Until(j.Due()).Round(time.Second).String() + "\nReason: " + j.Reason()
	}
	return ""
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
var _ scheduler.Runnable = &job{}

var _ scheduledJob = &job{}

type scheduledJob interface {
	Due() time.Time
	State() state
}

type job struct {
	action
	TadoClient
	*scheduler.Job
	logger *slog.Logger
}

func (j job) Run(_ context.Context) error {
	return j.action.Do(context.Background(), j.TadoClient, j.logger)
}
