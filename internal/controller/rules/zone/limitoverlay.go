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

var _ rules.Evaluator = LimitOverlayRule{}

// A LimitOverlayRule deletes a permanent overlay from a zone after a configured delay.
type LimitOverlayRule struct {
	zoneID   int
	zoneName string
	delay    time.Duration
	logger   *slog.Logger
}

func LoadLimitOverlay(id int, name string, cfg configuration.LimitOverlayConfiguration, _ poller.Update, logger *slog.Logger) (LimitOverlayRule, error) {
	return LimitOverlayRule{
		zoneID:   id,
		zoneName: name,
		delay:    cfg.Delay,
		logger:   logger.With(slog.String("rule", "limitOverlay")),
	}, nil
}

func (r LimitOverlayRule) Evaluate(update poller.Update) (action.Action, error) {
	a := action.Action{
		Label:  r.zoneName,
		Reason: "no manual temp setting detected",
		State: &State{
			homeId:   *update.HomeBase.Id,
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

	if zone.GetZoneOverlayTerminationType() == tado.ZoneOverlayTerminationTypeMANUAL {
		// If autoAway switched off the heating, this rule will reset that after r.delay. As a workaround, we only delete
		// the overlay if it's set to heating (temperature > 5ºC, i.e. "not off").
		if zone.GetTargetTemperature() > 5.0 {
			a.State.(*State).mode = action.ZoneInAutoMode
			a.Delay = r.delay
			a.Reason = "manual temp setting detected"
		}
	}
	r.logger.Debug("evaluated", slog.Any("result", a))

	return a, nil
}
