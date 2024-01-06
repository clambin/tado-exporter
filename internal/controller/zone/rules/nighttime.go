package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/tadotools"
	"time"
)

type NightTimeRule struct {
	zoneID    int
	zoneName  string
	timestamp configuration.Timestamp

	GetCurrentTime func() time.Time
}

func LoadNightTime(id int, name string, cfg configuration.NightTimeConfiguration, _ poller.Update) (NightTimeRule, error) {
	return NightTimeRule{
		zoneID:    id,
		zoneName:  name,
		timestamp: cfg.Timestamp,
	}, nil
}

var _ rules.Evaluator = NightTimeRule{}

func (n NightTimeRule) Evaluate(update poller.Update) (action.Action, error) {
	e := action.Action{Label: n.zoneName, Reason: "no manual temp setting detected"}
	s := State{
		zoneID:   n.zoneID,
		zoneName: n.zoneName,
		mode:     action.NoAction,
	}

	if state := tadotools.GetZoneState(update.ZoneInfo[n.zoneID]); state.Overlay == tado.PermanentOverlay && state.Heating() {
		// allow current time to be set during testing
		now := time.Now
		if n.GetCurrentTime != nil {
			now = n.GetCurrentTime
		}

		s.mode = action.ZoneInAutoMode
		e.Delay = getNextNightTimeDelay(now(), n.timestamp)
		e.Reason = "manual temp setting detected"
	}
	e.State = s
	return e, nil
}

func getNextNightTimeDelay(now time.Time, limit configuration.Timestamp) time.Duration {
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		limit.Hour, limit.Minutes, limit.Seconds, 0, time.Local)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}
