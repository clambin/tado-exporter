package rules

import (
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado"
	"slices"
	"strings"
	"time"
)

type Actions []Action

// GetNext takes the actions (determined by the active rules) and determines the action to take.
//
// GetNext prioritizes actions that lead to a state change (i.e. Action.Action == true).  Within those actions,
// it prioritizes actions that switch off a zone's heating over actions that move the zone to auto mode.
// If multiple actions of the same type exist, GetNext returns the earliest action.
//
// If GetNext finds no actions leading to a state change, it returns a single 'no action' Action, whose reason
// lists all conditions leading to the 'no action' action.
func (a Actions) GetNext() Action {
	if action, ok := a.getNextAction(); ok {
		return action
	}
	actions := a.getNoActionActions()
	if len(actions) == 0 {
		panic("no rules defined?")
	}
	return actions.getNoActionAction()
}

func (a Actions) getNextAction() (Action, bool) {
	// First, try to find the earliest action that switches heating off
	if action, ok := a.getFirstAction(func(action Action) bool {
		return action.Action && action.State.Overlay == tado.PermanentOverlay && !action.State.Heating()
	}); ok {
		return action, true
	}

	// Failing that, try to find the earliest action that switches the zone to auto mode
	if action, ok := a.getFirstAction(func(action Action) bool {
		return action.Action && action.State.Overlay == tado.NoOverlay
	}); ok {
		return action, true
	}

	// only called if len(a)>0 and the above are the only implemented states. So this should never happen
	return Action{}, false
}

func (a Actions) getFirstAction(eval func(s Action) bool) (Action, bool) {
	var minDelay time.Duration = -1
	var firstAction Action

	for _, action := range a {
		if eval(action) && (minDelay == -1 || action.Delay < minDelay) {
			firstAction = action
			minDelay = action.Delay
		}
	}
	return firstAction, minDelay != time.Duration(-1)
}

func (a Actions) getNoActionActions() Actions {
	actions := make(Actions, 0, len(a))
	for _, action := range a {
		if !action.Action {
			actions = append(actions, action)
		}
	}
	return actions
}

func (a Actions) getNoActionAction() Action {
	return Action{
		ZoneID:   a[0].ZoneID,
		ZoneName: a[0].ZoneName,
		Action:   false,
		Reason:   a.getCombinedReason(),
	}
}

func (a Actions) getCombinedReason() string {
	r := set.Create[string]()
	for _, action := range a {
		if action.Reason != "" {
			r.Add(action.Reason)
		}
	}
	uniqueReasons := r.List()
	slices.Sort(uniqueReasons)
	return strings.Join(uniqueReasons, ", ")
}
