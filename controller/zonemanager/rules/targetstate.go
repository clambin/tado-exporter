package rules

import (
	"github.com/clambin/tado-exporter/poller"
	"golang.org/x/exp/slog"
	"sort"
	"strings"
	"time"
)

var _ slog.LogValuer = &TargetState{}

type TargetState struct {
	ZoneID   int
	ZoneName string
	Action   bool
	State    poller.ZoneState
	Delay    time.Duration
	Reason   string
}

func (s TargetState) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("id", s.ZoneID),
		slog.String("name", s.ZoneName),
		slog.String("state", s.State.String()),
		slog.Duration("delay", s.Delay),
		slog.String("reason", s.Reason),
	)
}

type TargetStates []TargetState

func (t TargetStates) GetNextState() TargetState {
	if targetStates := t.filterTargetStates(true); len(targetStates) > 0 {
		return targetStates.getFirstAction()
	}
	targetStates := t.filterTargetStates(false)
	if len(targetStates) == 0 {
		panic("no rules defined?")
	}

	return targetStates.getNoAction()
}

func (t TargetStates) filterTargetStates(action bool) TargetStates {
	var targetStates TargetStates
	for _, targetState := range t {
		if targetState.Action == action {
			targetStates = append(targetStates, targetState)
		}
	}
	return targetStates
}

func (t TargetStates) getFirstAction() TargetState {
	targetState := t[0]
	for _, a := range t[1:] {
		if a.State == targetState.State {
			// same target state. take the targetState that fires the earliest
			if a.Delay < targetState.Delay {
				targetState = a
			}
		} else if a.State == poller.ZoneStateOff {
			// we give preference to rules that switch off the zone (i.e. currently the AutoAway rule)
			targetState = a
		}
	}
	return targetState
}

func (t TargetStates) getNoAction() TargetState {
	return TargetState{
		ZoneID:   t[0].ZoneID,
		ZoneName: t[0].ZoneName,
		Action:   false,
		State:    poller.ZoneStateUnknown,
		Delay:    0,
		Reason:   t.getCombinedReason(),
	}
}

func (t TargetStates) getCombinedReason() string {
	reasons := make(map[string]struct{})
	for _, targetState := range t {
		if targetState.Reason != "" {
			reasons[targetState.Reason] = struct{}{}
		}
	}
	var uniqueReasons []string
	for reason := range reasons {
		uniqueReasons = append(uniqueReasons, reason)
	}
	sort.Strings(uniqueReasons)
	return strings.Join(uniqueReasons, ", ")
}
