package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/poller"
)

type Evaluator struct {
	Config *ZoneConfig
	rules  []Rule
}

var _ Rule = &Evaluator{}

type Rule interface {
	Evaluate(*poller.Update) (TargetState, error)
}

func (e *Evaluator) Evaluate(update *poller.Update) (TargetState, error) {
	var targetState TargetState
	if err := e.load(update); err != nil {
		return targetState, err
	}

	targetStates := make(TargetStates, len(e.rules))
	for i, rule := range e.rules {
		next, err := rule.Evaluate(update)
		if err != nil {
			return targetState, err
		}
		targetStates[i] = next
	}

	next := targetStates.GetNextState()
	log(next, update)

	return next, nil
}

func (e *Evaluator) load(update *poller.Update) error {
	if len(e.rules) > 0 {
		return nil
	}

	zoneID, ok := update.GetZoneID(e.Config.Zone)
	if !ok {
		return fmt.Errorf("invalid zone found in config file: %s", e.Config.Zone)
	}

	for _, rawRule := range e.Config.Rules {
		switch rawRule.Kind {
		case AutoAway:
			e.rules = append(e.rules, &AutoAwayRule{
				ZoneID:   zoneID,
				ZoneName: e.Config.Zone,
				Delay:    rawRule.Delay,
				Users:    rawRule.Users,
			})
		case LimitOverlay:
			e.rules = append(e.rules, &LimitOverlayRule{
				zoneID:   zoneID,
				zoneName: e.Config.Zone,
				delay:    rawRule.Delay,
			})
		case NightTime:
			e.rules = append(e.rules, &NightTimeRule{
				zoneID:    zoneID,
				zoneName:  e.Config.Zone,
				timestamp: rawRule.Timestamp,
			})
		}
	}

	return nil
}
