package rules

import (
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/tado"
	"time"
)

type LimitOverlayRule struct {
	zoneID   int
	zoneName string
	delay    time.Duration
}

var _ Rule = &LimitOverlayRule{}

func (l *LimitOverlayRule) Evaluate(update *poller.Update) (TargetState, error) {
	next := TargetState{
		ZoneID:   l.zoneID,
		ZoneName: l.zoneName,
		Reason:   "no manual settings detected",
	}
	if state := tado.GetZoneState(update.ZoneInfo[l.zoneID]); state == tado.ZoneStateManual {
		next.Action = true
		next.State = tado.ZoneStateAuto
		next.Delay = l.delay
		next.Reason = "manual temp setting detected"
	}
	return next, nil
}
