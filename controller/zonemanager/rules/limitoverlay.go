package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
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
	if state := update.ZoneInfo[l.zoneID].GetState(); state == tado.ZoneStateManual {
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
