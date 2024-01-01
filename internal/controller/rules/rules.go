package rules

import (
	"cmp"
	"github.com/clambin/tado-exporter/internal/controller/rules/evaluate"
	"github.com/clambin/tado-exporter/internal/poller"
	"slices"
	"strings"
)

type Rules []evaluate.Evaluator

func (r Rules) Evaluate(update poller.Update) (evaluate.Evaluation, error) {
	actions := make([]evaluate.Evaluation, 0, len(r))
	noActions := make([]evaluate.Evaluation, 0, len(r))

	for _, _r := range r {
		e, err := _r.Evaluate(update)
		if err != nil {
			return evaluate.Evaluation{}, err
		}
		if e.IsAction() {
			actions = append(actions, e)
		} else {
			noActions = append(noActions, e)
		}
	}

	if len(actions) == 0 {
		return evaluate.Evaluation{Reason: getCombinedReason(noActions)}, nil
	}

	slices.SortFunc(actions, func(a, b evaluate.Evaluation) int {
		return cmp.Compare(a.Delay, b.Delay)
	})

	return actions[0], nil
}

func getCombinedReason(evaluations []evaluate.Evaluation) string {
	reasons := make(map[string]struct{})
	for _, e := range evaluations {
		reasons[e.Reason] = struct{}{}
	}
	results := make([]string, 0, len(reasons))
	for r := range reasons {
		results = append(results, r)
	}
	slices.Sort(results)
	return strings.Join(results, ", ")
}
