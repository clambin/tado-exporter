package zone

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/zone/notifier"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"log/slog"
	"sync"
)

type Controller struct {
	evaluator    rules.Evaluator
	task         *Task
	tadoClient   rules.TadoSetter
	notifiers    notifier.Notifiers
	poller       poller.Poller
	logger       *slog.Logger
	notification chan struct{}
	lock         sync.RWMutex
}

func New(tadoClient rules.TadoSetter, p poller.Poller, bot notifier.SlackSender, cfg rules.ZoneConfig, logger *slog.Logger) *Controller {
	controller := Controller{
		evaluator:    rules.Evaluator{Config: &cfg},
		tadoClient:   tadoClient,
		notifiers:    notifier.Notifiers{&notifier.SLogNotifier{Logger: logger}},
		poller:       p,
		notification: make(chan struct{}, 1),
		logger:       logger,
	}

	if bot != nil {
		controller.notifiers = append(controller.notifiers, &notifier.SlackNotifier{Slack: bot})
	}
	return &controller
}

func (c *Controller) Run(ctx context.Context) error {
	c.logger.Debug("started")
	defer c.logger.Debug("stopped")
	ch := c.poller.Subscribe()
	defer c.poller.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			if err := c.processUpdate(ctx, update); err != nil {
				c.logger.Error("failed to process tado update", "err", err, "zone", c.evaluator.Config.Zone)
			}
		case <-c.notification:
			if err := c.processResult(); err != nil {
				c.logger.Error("failed to set next state", "err", err, "zone", c.evaluator.Config.Zone)
			}
		}
	}
}

func (c *Controller) processUpdate(ctx context.Context, update poller.Update) error {
	next, err := c.evaluator.Evaluate(update)
	if err != nil {
		return fmt.Errorf("failed to evaluate rules: %w", err)
	}

	if next.Action {
		c.logger.Debug("scheduling job", "next", next, "zoneConfig", zoneLogger(update.ZoneInfo[next.ZoneID]))
		c.scheduleJob(ctx, next)
	} else {
		c.cancelJob(next)
	}

	return nil
}

func (c *Controller) scheduleJob(ctx context.Context, next rules.Action) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// if the same state is already scheduled for an earlier time, don't schedule it again.
	if c.task != nil {
		if c.task.nextState.State == next.State &&
			c.task.firesNoLaterThan(next) {
			return
		}

		// we will replace the running job, so cancel the old one
		c.task.job.Cancel()
	}

	c.task = newTask(ctx, c.tadoClient, next, c.notification)

	if next.Delay > 0 {
		c.notifiers.Notify(notifier.Queued, next)
	}
}

func (c *Controller) cancelJob(next rules.Action) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.task != nil {
		nextState := c.task.nextState
		nextState.Reason = next.Reason
		c.task.job.Cancel()
		c.notifiers.Notify(notifier.Canceled, nextState)
	}
}

func (c *Controller) processResult() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.task == nil {
		return nil
	}

	completed, err := c.task.job.Result()
	if !completed {
		return nil
	}

	if err == nil {
		c.notifiers.Notify(notifier.Done, c.task.nextState)
	} else if errors.Is(err, scheduler.ErrCanceled) {
		err = nil
	}

	c.task = nil
	return err
}

func (c *Controller) GetScheduled() (rules.Action, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.task != nil {
		return c.task.nextState, true
	}
	return rules.Action{}, false
}

func (c *Controller) ReportTask() (string, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.task != nil {
		return c.task.Report(), true
	}
	return "", false
}
