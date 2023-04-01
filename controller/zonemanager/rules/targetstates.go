package rules

import (
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado"
	"sort"
	"strings"
)

type TargetStates []TargetState

func (t TargetStates) GetNextState() TargetState {
	if targetStates := t.filterTargetStates(true); len(targetStates) > 0 {
		return targetStates.getAction()
	}
	targetStates := t.filterTargetStates(false)
	if len(targetStates) == 0 {
		panic("no rules defined?")
	}
	return targetStates.getNoAction()
}

func (t TargetStates) filterTargetStates(action bool) TargetStates {
	targetStates := make(TargetStates, 0, len(t))
	for _, targetState := range t {
		if targetState.Action == action {
			targetStates = append(targetStates, targetState)
		}
	}
	return targetStates
}

func (t TargetStates) getAction() TargetState {
	targetStates := t.getHeatingActions(false)
	if len(targetStates) == 0 {
		targetStates = t.getHeatingActions(true)
	}
	return targetStates[0]
}

func (t TargetStates) getHeatingActions(heating bool) TargetStates {
	targetStates := make(TargetStates, 0, len(t))
	for _, targetState := range t {
		if !heating {
			if targetState.State.Overlay == tado.PermanentOverlay && !targetState.State.Heating() {
				targetStates = append(targetStates, targetState)
			}
		} else {
			if targetState.State.Overlay == tado.NoOverlay {
				targetStates = append(targetStates, targetState)
			}
		}
	}
	sort.Slice(targetStates, func(i, j int) bool {
		return targetStates[i].Delay < targetStates[j].Delay
	})
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
