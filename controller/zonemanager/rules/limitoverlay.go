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

func (l *LimitOverlayRule) Evaluate(update *poller.Update) (NextState, error) {
	var next NextState
	if state := tado.GetZoneState(update.ZoneInfo[l.zoneID]); state == tado.ZoneStateManual {
		next = NextState{
			ZoneID:       l.zoneID,
			ZoneName:     l.zoneName,
			State:        tado.ZoneStateAuto,
			Delay:        l.delay,
			ActionReason: "manual temp setting detected",
			CancelReason: "room no longer in manual temp setting",
		}
	}
	return next, nil
}
