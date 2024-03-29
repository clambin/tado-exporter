package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"log/slog"
	"time"
)

type NightTimeRule struct {
	zoneID    int
	zoneName  string
	timestamp configuration.Timestamp
	logger    *slog.Logger

	GetCurrentTime func() time.Time
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

	if !update.Home {
		a.Reason = "home in AWAY mode"
		return a, nil
	}

	if state := tadotools.GetZoneState(update.ZoneInfo[r.zoneID]); state.Overlay == tado.PermanentOverlay && state.Heating() {
		// allow current time to be set during testing
		now := time.Now
		if r.GetCurrentTime != nil {
			now = r.GetCurrentTime
		}

		a.Delay = getNextNightTimeDelay(now(), r.timestamp)
		a.Reason = "manual temp setting detected"
		a.State.(*State).mode = action.ZoneInAutoMode
	}

	r.logger.Debug("evaluated",
		slog.Bool("home", bool(update.Home)),
		slog.Any("result", a),
	)

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
