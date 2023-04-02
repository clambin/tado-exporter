package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/poller"
	"golang.org/x/exp/slog"
)

type Evaluator struct {
	Config *ZoneConfig
	rules  []Rule
}

var _ Rule = &Evaluator{}

type Rule interface {
	Evaluate(*poller.Update) (Action, error)
}

func (e *Evaluator) Evaluate(update *poller.Update) (Action, error) {
	var action Action
	if err := e.load(update); err != nil {
		return action, err
	}

	actions := make(Actions, len(e.rules))
	for i, rule := range e.rules {
		next, err := rule.Evaluate(update)
		if err != nil {
			return action, err
		}
		actions[i] = next
	}

	next := actions.GetNext()

	slog.Debug("next state evaluated",
		"next", next,
		"zoneInfo", zoneInfo(update.ZoneInfo[next.ZoneID]),
		"devices", update.UserInfo,
	)

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
