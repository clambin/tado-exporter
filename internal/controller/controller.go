package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"log/slog"
	"sync/atomic"
	"time"
)

// A Controller evaluates all rules for a given home or zone. It receives updates from a Poller, evaluates all rules
// and executes the required action. If the required action has a configured delay, it schedules a job and manages its lifetime.
//
// Controller uses a Notifier (which may be a list of Notifiers) to inform the user.
type Controller struct {
	GroupEvaluator
	Publisher[poller.Update]
	TadoClient
	Notifier
	logger       *slog.Logger
	jobCompleted chan struct{}
	scheduledJob atomic.Pointer[job]
}

// NewController creates a new controller for the provided rules.
func NewController(rules GroupEvaluator, p Publisher[poller.Update], client TadoClient, notifier Notifier, l *slog.Logger) *Controller {
	return &Controller{
		GroupEvaluator: rules,
		Publisher:      p,
		TadoClient:     client,
		Notifier:       notifier,
		logger:         l,
		jobCompleted:   make(chan struct{}),
	}
}

// Run registers with a Poller and evaluates an incoming Update against its rules.
func (c *Controller) Run(ctx context.Context) error {
	ch := c.Publisher.Subscribe()
	defer c.Publisher.Unsubscribe(ch)

	c.logger.Debug("controller starting")
	defer c.logger.Debug("controller stopping")

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			if action, ok := c.processUpdate(update); ok {
				c.scheduleJob(ctx, action)
			} else {
				c.cancelJob()
			}
		case <-c.jobCompleted:
			c.processCompletedJob()
		}
	}
}

// processUpdate processes the update, evaluating its rules. If the outcome differs from the current state
// (as determined by the Update), it returns the action and true. Otherwise, it returns false.
func (c *Controller) processUpdate(update poller.Update) (Action, bool) {
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

// scheduleJob is called when processUpdate returns a new Action. It executes (or schedules) the required action.
func (c *Controller) scheduleJob(ctx context.Context, action Action) {
	// if a job is scheduled with the same action, but an earlier scheduled time, don't schedule a new job
	if c.scheduledJob.Load() != nil {
		if c.scheduledJob.Load().GetState() == action.GetState() && time.Now().Add(action.GetDelay()).After(c.scheduledJob.Load().Due()) {
			// c.logger.Debug("job for the same state already scheduled. not scheduling new job", "state", action.GetState(), "reason", action.GetReason())
			return
		}
	}

	// scheduling a new job. cancel any old one.
	c.cancelJob()

	// immediate action
	if action.GetDelay() == 0 {
		_ = c.doAction(ctx, action)
		return
	}

	// deferred action.
	j := job{
		TadoClient: c.TadoClient,
		Action:     action,
		Job: scheduler.Schedule(ctx, scheduler.RunFunc(func(ctx context.Context) error {
			return c.doAction(ctx, action)
		}), action.GetDelay(), c.jobCompleted),
	}
	c.scheduledJob.Store(&j)
	c.Notifier.Notify(action.Description(true))

}

// doAction executes the action and reports the result to the user through a Notifier.
// This is called either directly from scheduleJob, or from the scheduler once the Delay has passed.
func (c *Controller) doAction(ctx context.Context, action Action) error {
	if err := action.Do(ctx, c.TadoClient); err != nil {
		c.logger.Error("failed to execute action", "action", action, "err", err)
		return err
	}
	c.Notifier.Notify(action.Description(false))
	return nil
}

// cancelJob cancels any scheduled job.
func (c *Controller) cancelJob() {
	if j := c.scheduledJob.Load(); j != nil {
		j.Cancel()
	}
}

// processCompletedJob is notified by the scheduler once the job has completed and informs the user through a Notifier.
func (c *Controller) processCompletedJob() {
	if j := c.scheduledJob.Load(); j != nil {
		if _, err := j.Result(); err != nil {
			c.logger.Error("scheduled job failed", "err", err)
			return
		}
		c.Notifier.Notify(j.Description(false))
		c.scheduledJob.Store(nil)
	}
}

func (c *Controller) ReportTask() string {
	if j := c.scheduledJob.Load(); j != nil {
		return j.Description(true)
	}
	return ""
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ scheduler.Runnable = &job{}

type job struct {
	Action
	TadoClient
	*scheduler.Job
}

func (j job) Run(_ context.Context) error {
	return j.Action.Do(context.Background(), j.TadoClient)
}
