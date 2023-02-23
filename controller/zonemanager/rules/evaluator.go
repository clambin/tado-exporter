package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/tado"
	"time"
)

type Evaluator struct {
	Config *ZoneConfig
	rules  []Rule
}

type Rule interface {
	Evaluate(*poller.Update) (NextState, error)
}

var _ Rule = &Evaluator{}

type NextState struct {
	ZoneID       int
	ZoneName     string
	State        tado.ZoneState
	Delay        time.Duration
	ActionReason string
	CancelReason string
}

func (s NextState) IsZero() bool {
	return s.ZoneID == 0 || s.ZoneName == ""
}

func (e *Evaluator) Evaluate(update *poller.Update) (NextState, error) {
	if err := e.load(update); err != nil {
		return NextState{}, err
	}

	actions := make([]NextState, 0, len(e.rules))
	for _, rule := range e.rules {
		next, err := rule.Evaluate(update)
		if err != nil {
			return NextState{}, err
		}
		if !next.IsZero() {
			actions = append(actions, next)
		}
	}
	if len(actions) == 0 {
		return NextState{}, nil
	}

	action := actions[0]
	for _, a := range actions[1:] {
		if a.State == action.State {
			// same target state. take the action that fires the earliest
			if a.Delay < action.Delay {
				action = a
			}
		} else if a.State == tado.ZoneStateOff {
			// we give preference to rules that switch off the zone (i.e. currently the AutoAway rule)
			action = a
		}
	}
	return action, nil
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
				zoneID:   zoneID,
				zoneName: e.Config.Zone,
				delay:    rawRule.Delay,
				users:    rawRule.Users,
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
