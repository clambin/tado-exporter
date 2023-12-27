package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"time"
)

type LimitOverlayRule struct {
	zoneID   int
	zoneName string
	delay    time.Duration
}

var _ Rule = &LimitOverlayRule{}

func (l *LimitOverlayRule) Evaluate(update poller.Update) (Action, error) {
	next := Action{
		ZoneID:   l.zoneID,
		ZoneName: l.zoneName,
		Reason:   "no manual settings detected",
	}
	state := GetZoneState(update.ZoneInfo[l.zoneID])
	if !state.Home {
		next.Reason = "device is in away mode"
	} else if state.Overlay == tado.PermanentOverlay && state.Heating() {
		next.Action = true
		next.State = ZoneState{Overlay: tado.NoOverlay}
		next.Delay = l.delay
		next.Reason = "manual temp setting detected"
	}
	return next, nil
}
