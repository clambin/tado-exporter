package controller

import (
	"context"
	"errors"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"log/slog"
	"sync/atomic"
	"time"
)

// A groupController evaluates all rules for a given home or zone. It receives updates from a Poller, evaluates all rules
// and executes the required action. If the required action has a configured delay, it schedules a job and manages its lifetime.
//
// groupController uses a Notifier (which may be a list of Notifiers) to inform the user.
type groupController struct {
	groupEvaluator
	Publisher[poller.Update]
	TadoClient
	Notifier
	logger       *slog.Logger
	jobCompleted chan struct{}
	scheduledJob atomic.Pointer[job]
}

// newGroupController creates a new controller for the provided rules.
func newGroupController(rules groupEvaluator, p Publisher[poller.Update], client TadoClient, notifier Notifier, l *slog.Logger) *groupController {
	return &groupController{
		groupEvaluator: rules,
		Publisher:      p,
		TadoClient:     client,
		Notifier:       notifier,
		logger:         l,
		jobCompleted:   make(chan struct{}),
	}
}

// Run registers with a Poller and evaluates an incoming update against its rules.
func (c *groupController) Run(ctx context.Context) error {
	ch := c.Publisher.Subscribe()
	defer c.Publisher.Unsubscribe(ch)

	c.logger.Debug("group controller starting")
	defer c.logger.Debug("group controller stopping")

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			if a, ok := c.processUpdate(update); ok {
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
func (c *groupController) processUpdate(update poller.Update) (action, bool) {
	current, err := c.ParseUpdate(update)
	if err != nil {
		c.logger.Error("failed to parse update", "err", err)
		return nil, false
	}
	a, err := c.Evaluate(updateFromPollerUpdate(update))
	if err != nil {
		c.logger.Error("failed to evaluate zone rules", "err", err)
		return nil, false
	}
	return a, a.GetState() != current.GetState()
}

// scheduleJob is called when processUpdate returns a new action. It executes (or schedules) the required action.
func (c *groupController) scheduleJob(ctx context.Context, action action) {
	// if a job is scheduled with the same action, but an earlier scheduled time, don't schedule a new job
	j := c.scheduledJob.Load()
	if j != nil {
		if j.GetState() == action.GetState() {
			// truncate old & new due times up to a minute and only start a new job (canceling the old one) if newDue is after due.
			// this avoids canceling the current job & immediately scheduling a new one if the old & new due times are very close
			// (e.g. in case of a rule like nighttime, which targets a specific time of day.
			due := j.Due().Truncate(time.Minute)
			newDue := time.Now().Local().Add(action.GetDelay()).Truncate(time.Minute)
			if !newDue.Before(due) {
				// c.logger.Debug("job for the same state already scheduled. not scheduling new job", "state", action.GetState(), "reason", action.GetReason())
				return
			}
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

// doAction executes the action and reports the result to the user through a Notifier.
// This is called either directly from scheduleJob, or from the scheduler once the Delay has passed.
func (c *groupController) doAction(ctx context.Context, action action) error {
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
func (c *groupController) cancelJob(a action) {
	if j := c.scheduledJob.Load(); j != nil {
		j.Cancel()
		if c.Notifier != nil {
			c.Notifier.Notify(j.Description(true) + " canceled\nReason: " + a.GetReason())
		}
	}
}

// processCompletedJob is notified by the scheduler once the job has completed and informs the user through a Notifier.
func (c *groupController) processCompletedJob() {
	if j := c.scheduledJob.Load(); j != nil {
		defer c.scheduledJob.Store(nil)
		_, err := j.Result()
		if err != nil && !errors.Is(err, context.Canceled) {
			c.logger.Error("scheduled job failed", "err", err)
			return
		}
	}
}

func (c *groupController) ReportTask() string {
	if j := c.scheduledJob.Load(); j != nil {
		return j.Description(false) + " in " + time.Until(j.Due()).Truncate(time.Second).String()
	}
	return ""
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ scheduler.Runnable = &job{}

type job struct {
	action
	TadoClient
	*scheduler.Job
}

func (j job) Run(_ context.Context) error {
	return j.action.Do(context.Background(), j.TadoClient)
}
