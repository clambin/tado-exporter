package rules

import (
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado"
	"sort"
	"strings"
	"time"
)

type TargetStates []TargetState

func (t TargetStates) GetNextState() TargetState {
	if targetState, ok := t.getAction(); ok {
		return targetState
	}
	targetStates := t.getNoActionTargetStates()
	if len(targetStates) == 0 {
		panic("no rules defined?")
	}
	return targetStates.getNoAction()
}

func (t TargetStates) getAction() (TargetState, bool) {
	// First, try to find the earliest action that switches heating off
	if targetState, ok := t.getFirstAction(func(targetState TargetState) bool {
		return targetState.Action && targetState.State.Overlay == tado.PermanentOverlay && !targetState.State.Heating()
	}); ok {
		return targetState, true
	}

	// Failing that, try to find the earliest action that switches the zone to auto mode
	if targetState, ok := t.getFirstAction(func(targetState TargetState) bool {
		return targetState.Action && targetState.State.Overlay == tado.NoOverlay
	}); ok {
		return targetState, true
	}

	// only called if len(t)>0 and the above are the only implemented states. So this should never happen
	return TargetState{}, false
}

func (t TargetStates) getFirstAction(eval func(s TargetState) bool) (TargetState, bool) {
	var minDelay time.Duration = -1
	var firstTargetState TargetState

	for _, targetState := range t {
		if eval(targetState) && (minDelay == -1 || targetState.Delay < minDelay) {
			firstTargetState = targetState
			minDelay = targetState.Delay
		}
	}
	return firstTargetState, minDelay != time.Duration(-1)
}

func (t TargetStates) getNoActionTargetStates() TargetStates {
	targetStates := make(TargetStates, 0, len(t))
	for _, targetState := range t {
		if !targetState.Action {
			targetStates = append(targetStates, targetState)
		}
	}
	return targetStates
}

func (t TargetStates) getNoAction() TargetState {
	return TargetState{
		ZoneID:   t[0].ZoneID,
		ZoneName: t[0].ZoneName,
		Action:   false,
		Reason:   t.getCombinedReason(),
	}
}

func (t TargetStates) getCombinedReason() string {
	r := set.Create[string]()
	for _, targetState := range t {
		if targetState.Reason != "" {
			r.Add(targetState.Reason)
		}
	}
	uniqueReasons := r.List()
	sort.Strings(uniqueReasons)
	return strings.Join(uniqueReasons, ", ")
}
