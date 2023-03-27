package rules

import (
	"github.com/clambin/tado-exporter/poller"
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
	if state := poller.GetZoneState(update.ZoneInfo[l.zoneID]); state == poller.ZoneStateManual {
		next.Action = true
		next.State = poller.ZoneStateAuto
		next.Delay = l.delay
		next.Reason = "manual temp setting detected"
	}
	return next, nil
}
