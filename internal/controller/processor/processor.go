package processor

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"log/slog"
	"sync"
)

// A Processor receives updated from a Poller and evaluates a set of rules.  If a rule results in an action,
// Processor schedules a task and manages that task until its completed.
//
// The Controller uses this to evaluate rules for a home and its zones.
type Processor struct {
	loader     RulesLoader
	rules      rules.Evaluator
	task       *Task
	tadoClient action.TadoClient
	notifiers  notifier.Notifiers
	poller     poller.Poller
	logger     *slog.Logger
	taskDone   chan struct{}
	lock       sync.RWMutex
}

type RulesLoader func(update poller.Update) (rules.Evaluator, error)

func New(tadoClient action.TadoClient, p poller.Poller, bot notifier.SlackSender, loader RulesLoader, logger *slog.Logger) *Processor {
	processor := Processor{
		loader:     loader,
		tadoClient: tadoClient,
		poller:     p,
		logger:     logger,
		notifiers:  notifier.Notifiers{&notifier.SLogNotifier{Logger: logger}},
		taskDone:   make(chan struct{}, 1),
	}

	if bot != nil {
		processor.notifiers = append(processor.notifiers, &notifier.SlackNotifier{Slack: bot})
	}
	return &processor
}

func (p *Processor) Run(ctx context.Context) error {
	p.logger.Debug("started")
	defer p.logger.Debug("stopped")
	ch := p.poller.Subscribe()
	defer p.poller.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			a, err := p.Evaluate(update)
			if err != nil {
				p.logger.Error("failed to get next action", "err", err)
				break
			}
			p.processUpdate(ctx, a)
		case <-p.taskDone:
			if err := p.processResult(); err != nil {
				p.logger.Error("failed to set next state", "err", err)
			}
		}
	}
}

func (p *Processor) Evaluate(update poller.Update) (action action.Action, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.rules == nil {
		if p.rules, err = p.loader(update); err != nil {
			return action, fmt.Errorf("failed to load rules: %w", err)
		}
	}

	return p.rules.Evaluate(update)
}

func (p *Processor) processUpdate(ctx context.Context, action action.Action) {
	if action.IsAction() {
		p.logger.Debug("scheduling job", "next", action)
		p.scheduleJob(ctx, action)
	} else {
		p.cancelJob(action)
	}
}

func (p *Processor) scheduleJob(ctx context.Context, action action.Action) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// if the same state is already scheduled for an earlier time, don't schedule it again.
	if p.task != nil {
		if p.task.action.State.IsEqual(action.State) && p.task.scheduledBefore(action) {
			return
		}

		// we will replace the running job, so cancel the old one
		p.task.job.Cancel()
	}

	p.task = newTask(ctx, p.tadoClient, action, p.taskDone)

	if action.Delay > 0 {
		p.notifiers.Notify(notifier.Queued, action)
	}
}

func (p *Processor) cancelJob(a action.Action) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.task != nil {
		canceledAction := p.task.action
		canceledAction.Reason = a.Reason
		p.task.job.Cancel()
		p.notifiers.Notify(notifier.Canceled, canceledAction)
	}
}

func (p *Processor) processResult() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.task == nil {
		return nil
	}

	completed, err := p.task.job.Result()
	if !completed {
		return nil
	}

	if err == nil {
		p.notifiers.Notify(notifier.Done, p.task.action)
	} else if errors.Is(err, scheduler.ErrCanceled) {
		err = nil
	}

	p.task = nil
	return err
}

func (p *Processor) ReportTask() (string, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.task == nil {
		return "", false
	}
	return p.task.Report(), true
}
