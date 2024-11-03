package zone

import (
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"log/slog"
	"time"
)

// A NightTimeRule deletes a manual overlay at a specific time of day.
type NightTimeRule struct {
	zoneID         int
	zoneName       string
	timestamp      configuration.Timestamp
	logger         *slog.Logger
	getCurrentTime func() time.Time
}

func LoadNightTime(id int, name string, cfg configuration.NightTimeConfiguration, _ poller.Update, logger *slog.Logger) (NightTimeRule, error) {
	return NightTimeRule{
		zoneID:    id,
		zoneName:  name,
		timestamp: cfg.Timestamp,
		logger:    logger.With(slog.String("rule", "nightTime")),
	}, nil
}

var _ rules.Evaluator = NightTimeRule{}

func (r NightTimeRule) Evaluate(update poller.Update) (action.Action, error) {
	a := action.Action{
		Label:  r.zoneName,
		Reason: "no manual temp setting detected",
		State: &State{
			zoneID:   r.zoneID,
			zoneName: r.zoneName,
			mode:     action.NoAction,
		},
	}

	if !update.Home() {
		a.Reason = "home in AWAY mode"
		return a, nil
	}

	zone, err := update.GetZone(r.zoneName)
	if err != nil {
		return a, err
	}

	// If autoAway switched off the heating, this rule will reset that after r.delay. As a workaround, we only delete
	// the overlay if it's set to heating (temperature > 5ºC, i.e. "not off").
	if zone.GetZoneOverlayTerminationType() == tado.ZoneOverlayTerminationTypeMANUAL && zone.GetTargetTemperature() > 5.0 {
		// allow current time to be set during testing
		now := time.Now
		if r.getCurrentTime != nil {
			now = r.getCurrentTime
		}

		a.Delay = getNextNightTimeDelay(now(), r.timestamp)
		a.Reason = "manual temp setting detected"
		a.State.(*State).mode = action.ZoneInAutoMode
	}

	r.logger.Debug("evaluated", slog.Any("result", a))
	return a, nil
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
