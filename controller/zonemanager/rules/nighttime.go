package rules

import (
	"github.com/clambin/tado-exporter/poller"
	"time"
)

type NightTimeRule struct {
	zoneID    int
	zoneName  string
	timestamp Timestamp
}

var _ Rule = &NightTimeRule{}

var testForceTime time.Time

func (n *NightTimeRule) Evaluate(update *poller.Update) (TargetState, error) {
	next := TargetState{
		ZoneID:   n.zoneID,
		ZoneName: n.zoneName,
		Reason:   "no manual settings detected",
	}

	if state := poller.GetZoneState(update.ZoneInfo[n.zoneID]); state == poller.ZoneStateManual {
		now := time.Now()
		if !testForceTime.IsZero() {
			now = testForceTime
		}
		next.Action = true
		next.State = poller.ZoneStateAuto
		next.Delay = getNextNightTimeDelay(now, n.timestamp)
		next.Reason = "manual temp setting detected"
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
