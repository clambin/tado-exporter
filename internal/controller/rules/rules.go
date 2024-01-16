package rules

import (
	"cmp"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/poller"
	"slices"
	"strings"
)

type Evaluator interface {
	Evaluate(update poller.Update) (action.Action, error)
}

type Rules []Evaluator

func (r Rules) Evaluate(update poller.Update) (action.Action, error) {
	actions := make([]action.Action, 0, len(r))
	noActions := make([]action.Action, 0, len(r))

	for _, _r := range r {
		e, err := _r.Evaluate(update)
		if err != nil {
			return action.Action{}, err
		}
		if e.IsAction() {
			actions = append(actions, e)
		} else {
			noActions = append(noActions, e)
		}
	}

	if len(actions) == 0 {
		return action.Action{Reason: getCombinedReason(noActions)}, nil
	}

	actions = filterFirstAction(actions)
	actions[0].Reason = getCombinedReason(actions)

	return actions[0], nil
}

func filterFirstAction(actions []action.Action) []action.Action {
	slices.SortFunc(actions, func(a, b action.Action) int { return cmp.Compare(a.Delay, b.Delay) })

	// only keep actions that result in the same state change
	index := 1
	for _, a := range actions[1:] {
		if a.State != nil && a.State.Mode() != actions[0].State.Mode() {
			break
		}
		index++
	}
	return slices.Delete(actions, index, len(actions))
}

func getCombinedReason(actions []action.Action) string {
	slices.SortFunc(actions, func(a, b action.Action) int { return cmp.Compare(a.Reason, b.Reason) })

	results := make([]string, 0, len(actions))
	var last string

	for _, a := range actions {
		// TODO: a.Reason != "" is an effort to remove "home in HOME mode" where it's not really needed.
		// Not sure if this is the best approach.
		if a.Reason != "" && a.Reason != last {
			results = append(results, a.Reason)
			last = a.Reason
		}
	}

	return strings.Join(results, ", ")
}
