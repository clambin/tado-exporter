package rules

import (
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/tado"
	"time"
)

type NightTimeRule struct {
	zoneID    int
	zoneName  string
	timestamp Timestamp
}

var _ Rule = &NightTimeRule{}

var testForceTime time.Time

func (n *NightTimeRule) Evaluate(update *poller.Update) (NextState, error) {
	var next NextState
	if state := tado.GetZoneState(update.ZoneInfo[n.zoneID]); state == tado.ZoneStateManual {

		now := time.Now()
		if !testForceTime.IsZero() {
			now = testForceTime
		}

		next = NextState{
			ZoneID:       n.zoneID,
			ZoneName:     n.zoneName,
			State:        tado.ZoneStateAuto,
			Delay:        getNextNightTimeDelay(now, n.timestamp),
			ActionReason: "manual temp setting detected",
			CancelReason: "room no longer in manual temp setting",
		}
	}
	return next, nil
}

func getNextNightTimeDelay(now time.Time, limit Timestamp) time.Duration {
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		limit.Hour, limit.Minutes, limit.Seconds, 0, time.Local)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}
