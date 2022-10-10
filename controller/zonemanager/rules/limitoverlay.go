package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
)

type LimitOverlayRule struct {
	zoneID   int
	zoneName string
	config   *configuration.ZoneLimitOverlay
}

var _ Rule = &LimitOverlayRule{}

func (l *LimitOverlayRule) Evaluate(update *poller.Update) (*NextState, error) {
	if state := update.ZoneInfo[l.zoneID].GetState(); state != tado.ZoneStateManual {
		return nil, nil
	}
	return &NextState{
		ZoneID:       l.zoneID,
		ZoneName:     l.zoneName,
		State:        tado.ZoneStateAuto,
		Delay:        l.config.Delay,
		ActionReason: "manual temp setting detected",
		CancelReason: "room no longer in manual temp setting",
	}, nil
}
