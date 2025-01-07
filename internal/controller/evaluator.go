package controller

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
	"sync/atomic"
	"time"
)

// A groupEvaluator evaluates all rules for a given home or zone. It receives updates from a Poller, evaluates all rules
// and executes the required action. If the required action has a configured delay, it schedules a job and manages its lifetime.
//
// groupEvaluator uses a Notifier (which may be a list of Notifiers) to inform the user.
type groupEvaluator struct {
	Publisher[poller.Update]
	TadoClient
	notifier.Notifier
	logger       *slog.Logger
	queuedAction atomic.Value
	rules.Rules
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
		Publisher:  p,
		TadoClient: client,
		Notifier:   notifier,
		logger:     l,
		Rules:      rules,
	}
}

// Run registers with a Poller and evaluates an incoming update against its rules.
func (g *groupEvaluator) Run(ctx context.Context) error {
	ch := g.Publisher.Subscribe()
	defer g.Publisher.Unsubscribe(ch)

	g.logger.Debug("group controller starting")
	defer g.logger.Debug("group controller stopping")

	queuedActionTicker := time.NewTicker(5 * time.Second)
	defer queuedActionTicker.Stop()

	for {
		g.processQueuedAction(ctx)
		select {
		case <-ctx.Done():
			return nil
		case u := <-ch:
			g.processUpdate(u)
		case <-queuedActionTicker.C:
		}
	}
}

// processUpdate processes the update, evaluating its rules and determining the required action.
// If the required action differs from the current state, or it needs to run before any currently queued action, it queues the action.
// Otherwise, any queued Action is canceled.
func (g *groupEvaluator) processUpdate(update poller.Update) {
	if g.Rules.Count() == 0 {
		return
	}

	newState, err := g.Rules.GetState(update)
	if err != nil {
		g.logger.Error("failed to parse update", "err", err)
		return
	}

	action, err := g.Rules.Evaluate(newState)
	if err != nil {
		g.logger.Error("failed to evaluate zone rules", "err", err)
		return
	}

	if !action.IsState(newState) {
		if g.shouldSchedule(action) {
			g.scheduleJob(action)
		}
	} else {
		g.cancelJob(action)
	}
}

// shouldSchedule returns true if the newAction should be scheduled, i.e. either the action is different from the queued action,
// or newAction should run before the scheduled action.
func (g *groupEvaluator) shouldSchedule(newAction rules.Action) bool {
	// check any action that is currently queued. if no action is queued, or it is for a different state,
	// then we need to queue the new action.
	queued, ok := g.getQueuedAction()
	if !ok || !queued.action.IsAction(newAction) {
		return true
	}
	// truncate old & new due times up to a minute and only start a new job (canceling the old one) if newDue is after due.
	// this avoids canceling the current job & immediately scheduling a new one if the old & new due times are very close
	// (e.g. in case of a rule like nighttime, which targets a specific time of day.
	due := queued.due.Truncate(time.Minute)
	newDue := time.Now().Add(newAction.Delay().Truncate(time.Minute))
	return newDue.Before(due)
}

// scheduleJob is called when processUpdate returns a new action. It executes (or schedules) the required action.
func (g *groupEvaluator) scheduleJob(action rules.Action) {
	// cancel any previously queued action
	queued, ok := g.getQueuedAction()
	if ok {
		g.cancelJob(queued.action)
	}

	// queued action
	queued = queuedAction{
		action: action,
		due:    time.Now().Add(action.Delay().Truncate(time.Minute)),
	}
	g.queuedAction.Store(queued)
	if g.Notifier != nil && action.Delay() != 0 {
		g.Notifier.Notify(action.Description(true) + "\nReason: " + action.Reason())
	}
}

// cancelJob cancels any scheduled job.
func (g *groupEvaluator) cancelJob(a rules.Action) {
	if queued, ok := g.getQueuedAction(); ok {
		queued.due = time.Time{}
		g.queuedAction.Store(queued)
		if g.Notifier != nil {
			g.Notifier.Notify(queued.action.Description(false) + " canceled\nReason: " + a.Reason())
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
	if err := queued.action.Do(ctx, g.TadoClient, g.logger); err != nil {
		g.logger.Error("failed to execute action", "action", queued.action, "err", err)
		return
	}
	// clear the queued action
	queued.due = time.Time{}
	g.queuedAction.Store(queued)
	// notify that the action has been run.
	if g.Notifier != nil {
		g.Notifier.Notify(queued.action.Description(false) + "\nReason: " + queued.action.Reason())
	}

}

func (g *groupEvaluator) ReportTask() string {
	if queued, ok := g.getQueuedAction(); ok {
		return queued.action.Description(false) + " in " + queued.action.Delay().Round(time.Second).String() + "\nReason: " + queued.action.Reason()
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
	action rules.Action
	due    time.Time
}
