package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

type Evaluator struct {
	Config *ZoneConfig
	rules  []Rule
}

var _ Rule = &Evaluator{}

type Rule interface {
	Evaluate(poller.Update) (Action, error)
}

func (e *Evaluator) Evaluate(update poller.Update) (Action, error) {
	var action Action
	if err := e.load(update); err != nil {
		return action, err
	}

	id, home, err := e.zoneInHomeMode(update)
	if err != nil {
		return action, err
	}
	if !home {
		action.ZoneID = id
		action.ZoneName = e.Config.Zone
		action.Reason = "device in away mode"
		return action, nil
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

func (e *Evaluator) load(update poller.Update) error {
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

func (e *Evaluator) zoneInHomeMode(update poller.Update) (int, bool, error) {
	zoneID, ok := update.GetZoneID(e.Config.Zone)
	if !ok {
		return 0, false, fmt.Errorf("invalid zone name: %s", e.Config.Zone)
	}

	info, ok := update.ZoneInfo[zoneID]
	if !ok {
		return 0, false, fmt.Errorf("missing zoneInfo for zone %s", e.Config.Zone)
	}

	return zoneID, info.TadoMode == "HOME", nil
}
