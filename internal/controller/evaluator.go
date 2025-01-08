package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
	"sync/atomic"
	"time"
)

// A groupEvaluator evaluates all rules for a given home or zone. It receives updates from a Poller, evaluates all rules
// and executes the required action. If the required action has a configured delay, it queues the action and manages its lifetime.
type groupEvaluator struct {
	publisher    Publisher[poller.Update]
	tadoClient   TadoClient
	notifier     notifier.Notifier
	logger       *slog.Logger
	queuedAction atomic.Value
	rules        rules.Rules
}

// newGroupEvaluator creates a new controller for the provided rules.
func newGroupEvaluator(
	rules rules.Rules,
	p Publisher[poller.Update],
	client TadoClient,
	notifier notifier.Notifier,
	l *slog.Logger,
) *groupEvaluator {
	return &groupEvaluator{
		publisher:  p,
		tadoClient: client,
		notifier:   notifier,
		logger:     l,
		rules:      rules,
	}
}

// Run registers with a Poller and evaluates an incoming update against its rules.
func (g *groupEvaluator) Run(ctx context.Context) error {
	ch := g.publisher.Subscribe()
	defer g.publisher.Unsubscribe(ch)

	g.logger.Debug("group controller starting")
	defer g.logger.Debug("group controller stopping")

	for {
		select {
		case <-ctx.Done():
			return nil
		case u := <-ch:
			if err := g.processUpdate(u); err != nil {
				g.logger.Error("failed to process update", "err", err)
			}
		case <-time.After(5 * time.Second):
		}
		g.processQueuedAction(ctx)
	}
}

// processUpdate processes the update, evaluating its rules and determining the required action.
// If the required action differs from the current state, or it needs to run before any currently queued action, it queues the action.
// Otherwise, any queued Action is canceled.
func (g *groupEvaluator) processUpdate(update poller.Update) error {
	newState, err := g.rules.GetState(update)
	if err != nil {
		return fmt.Errorf("failed to parse update: %w", err)
	}

	action, err := g.rules.Evaluate(newState)
	if err != nil {
		return fmt.Errorf("failed to evaluate zone rules: %w", err)
	}

	if !action.IsState(newState) {
		g.queueAction(action)
	} else {
		// we're already in the desired state. cancel any queued action
		g.cancelQueuedAction(action)
	}

	return nil
}

// queueAction queues a new action for execution.
func (g *groupEvaluator) queueAction(action rules.Action) {
	// should we perform this action?
	if !g.shouldSchedule(action) {
		return
	}

	// cancel any previously queued action
	if queued, ok := g.getQueuedAction(); ok {
		g.cancelQueuedAction(queued.Action)
	}

	// queue action
	delay := action.Delay().Truncate(time.Minute)
	g.logger.Debug("queueing action", "action", action, "delay", delay)
	g.queuedAction.Store(queuedAction{Action: action, due: time.Now().Add(delay)})

	// inform the user of the queued action, unless the action will be performed immediately (as that already inform the user).
	if g.notifier != nil && action.Delay() != 0 {
		g.notifier.Notify(action.Description(true) + "\nReason: " + action.Reason())
	}
}

// shouldSchedule checks if the newAction should be performed, i.e. either the new action is different from the queued action,
// or the new action should run before the queued action.
func (g *groupEvaluator) shouldSchedule(newAction rules.Action) bool {
	// check any action that is currently queued. if no action is queued, or it is for a different state,
	// then we need to queue the new action.
	queued, ok := g.getQueuedAction()
	if !ok || !queued.IsAction(newAction) {
		return true
	}
	// truncate old & new due times up to a minute before comparing them.
	// this avoids canceling the current job & immediately queuing a new one if the old & new due times are very close
	// (e.g. in case of a rule like nighttime, which targets a specific time of day).
	due := queued.due.Truncate(time.Minute)
	newDue := time.Now().Add(newAction.Delay().Truncate(time.Minute))
	return newDue.Before(due)
}

// cancelQueuedAction cancels any queued action.
func (g *groupEvaluator) cancelQueuedAction(a rules.Action) {
	if queued, ok := g.getQueuedAction(); ok {
		queued.due = time.Time{}
		g.queuedAction.Store(queued)
		if g.notifier != nil {
			g.notifier.Notify(queued.Description(false) + " canceled\nReason: " + a.Reason())
		}
	}
}

// processQueuedAction runs the queued action at the required time.
func (g *groupEvaluator) processQueuedAction(ctx context.Context) {
	queued, ok := g.getQueuedAction()
	// if no action is queued, or it's not due yet, do nothing
	if !ok || queued.due.After(time.Now()) {
		return
	}
	// perform the action
	if err := queued.Do(ctx, g.tadoClient, g.logger); err != nil {
		g.logger.Error("failed to execute action", "action", queued.Action, "err", err)
		return
	}
	g.logger.Debug("performed queued action", "action", queued.Action)
	// clear the queued action
	queued.due = time.Time{}
	g.queuedAction.Store(queued)
	// notify that the action has been run.
	if g.notifier != nil {
		g.notifier.Notify(queued.Description(false) + "\nReason: " + queued.Reason())
	}
}

func (g *groupEvaluator) ReportTask() string {
	if queued, ok := g.getQueuedAction(); ok {
		return queued.Description(false) + " in " + time.Until(queued.due).Round(time.Second).String() + "\nReason: " + queued.Reason()
	}
	return ""
}

func (g *groupEvaluator) getQueuedAction() (queuedAction, bool) {
	queued := g.queuedAction.Load()
	if queued == nil {
		return queuedAction{}, false
	}
	return queued.(queuedAction), !queued.(queuedAction).due.IsZero()
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// queuedAction indicates a action waiting to be run. if due is zero, then no action is queued.
type queuedAction struct {
	rules.Action
	due time.Time
}
