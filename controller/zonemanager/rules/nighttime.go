package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"time"
)

type NightTimeRule struct {
	zoneID   int
	zoneName string
	config   *configuration.ZoneNightTime
}

var _ Rule = &NightTimeRule{}

var testForceTime time.Time

func (n *NightTimeRule) Evaluate(update *poller.Update) (*NextState, error) {
	if state := update.ZoneInfo[n.zoneID].GetState(); state != tado.ZoneStateManual {
		return nil, nil
	}

	now := time.Now()
	if !testForceTime.IsZero() {
		now = testForceTime
	}

	return &NextState{
		ZoneID:       n.zoneID,
		ZoneName:     n.zoneName,
		State:        tado.ZoneStateAuto,
		Delay:        getNextNightTimeDelay(now, n.config.Time),
		ActionReason: "manual temp setting detected",
		CancelReason: "room no longer in manual temp setting",
	}, nil
}

func getNextNightTimeDelay(now time.Time, limit configuration.ZoneNightTimeTimestamp) time.Duration {
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		limit.Hour, limit.Minutes, limit.Seconds, 0, time.Local)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}
