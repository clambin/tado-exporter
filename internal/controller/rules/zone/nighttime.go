package zone

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/rules/evaluate"
	"github.com/clambin/tado-exporter/internal/poller"
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

var _ evaluate.Evaluator = NightTimeRule{}

func (n NightTimeRule) Evaluate(update poller.Update) (evaluate.Evaluation, error) {
	var e evaluate.Evaluation

	e.Reason = "no manual temp setting detected"
	if state := GetZoneState(update.ZoneInfo[n.zoneID]); state.Overlay == tado.PermanentOverlay && state.Heating() {
		// allow current time to be set during testing
		now := time.Now
		if n.GetCurrentTime != nil {
			now = n.GetCurrentTime
		}

		e.Do = func(ctx context.Context, setter evaluate.TadoSetter) error {
			return setter.DeleteZoneOverlay(ctx, n.zoneID)
		}
		e.Delay = getNextNightTimeDelay(now(), n.timestamp)
		e.Reason = "manual temp setting detected"
	}
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
