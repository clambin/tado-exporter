package rules

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"time"
)

type Evaluator struct {
	Config   *configuration.ZoneConfig
	ZoneID   int
	ZoneName string
	rules    []Rule
}

type Rule interface {
	Evaluate(*poller.Update) (*NextState, error)
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

func (e *Evaluator) Evaluate(update *poller.Update) (action *NextState, err error) {
	if err = e.load(update); err != nil {
		return nil, err
	}

	actions := make([]*NextState, 0, len(e.rules))
	for _, rule := range e.rules {
		var next *NextState
		if next, err = rule.Evaluate(update); err != nil {
			return nil, err
		}
		if next != nil {
			actions = append(actions, next)
		}
	}
	if len(actions) == 0 {
		return nil, nil
	}

	action = actions[0]
	for _, a := range actions[1:] {
		if a.State == action.State {
			// same target state. take the action that fires the earliest
			if a.Delay < action.Delay {
				action.Delay = a.Delay
				action.ActionReason = a.ActionReason
				action.CancelReason = a.CancelReason
			}
		} else if a.State == tado.ZoneStateOff {
			// we give preference to rules that switch off the zone (i.e. currently the AutoAway rule)
			action = a
		}
	}
	return
}

func (e *Evaluator) load(update *poller.Update) error {
	if len(e.rules) > 0 {
		return nil
	}

	var exists bool
	if e.ZoneID, e.ZoneName, exists = update.LookupZone(e.Config.ZoneID, e.Config.ZoneName); !exists {
		return fmt.Errorf("invalid zone found in config file: zoneID: %d, zoneName: %s", e.Config.ZoneID, e.Config.ZoneName)
	}

	if e.Config.LimitOverlay.Enabled {
		e.rules = append(e.rules, &LimitOverlayRule{
			zoneID:   e.ZoneID,
			zoneName: e.ZoneName,
			config:   &e.Config.LimitOverlay,
		})
	}

	if e.Config.NightTime.Enabled {
		e.rules = append(e.rules, &NightTimeRule{
			zoneID:   e.ZoneID,
			zoneName: e.ZoneName,
			config:   &e.Config.NightTime,
		})
	}

	if e.Config.AutoAway.Enabled {
		e.rules = append(e.rules, &AutoAwayRule{
			zoneID:   e.ZoneID,
			zoneName: e.ZoneName,
			config:   &e.Config.AutoAway,
		})
	}

	return nil
}
