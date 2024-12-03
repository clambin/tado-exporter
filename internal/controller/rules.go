package controller

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/poller"
)

// A rule takes the current update and determines the next action.
type rule interface {
	Evaluate(poller.Update) (action, error)
}

// Rules takes the current update, determines the next action for each rule and returns the first action required.
type rules []rule

func (r rules) Evaluate(u poller.Update) ([]action, error) {
	actions := make([]action, len(r))
	for i := range r {
		var err error
		actions[i], err = r[i].Evaluate(u)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule %d: %w", i+1, err)
		}
	}
	return actions, nil
}
