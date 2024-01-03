package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"time"
)

type LimitOverlayRule struct {
	zoneID   int
	zoneName string
	delay    time.Duration
}

func LoadLimitOverlay(id int, name string, cfg configuration.LimitOverlayConfiguration, _ poller.Update) (LimitOverlayRule, error) {
	return LimitOverlayRule{
		zoneID:   id,
		zoneName: name,
		delay:    cfg.Delay,
	}, nil
}

//var _ evaluate.Evaluator = LimitOverlayRule{}

func (l LimitOverlayRule) Evaluate(update poller.Update) (action.Action, error) {
	s := State{
		zoneID:   l.zoneID,
		zoneName: l.zoneName,
		mode:     action.NoAction,
	}
	e := action.Action{Label: l.zoneName, Reason: "no manual temp setting detected"}

	if state := GetZoneState(update.ZoneInfo[l.zoneID]); state.Overlay == tado.PermanentOverlay {
		s.mode = action.ZoneInAutoMode
		e.Delay = l.delay
		e.Reason = "manual temp setting detected"
	}
	e.State = s
	return e, nil
}
