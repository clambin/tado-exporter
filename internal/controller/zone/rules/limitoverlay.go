package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"log/slog"
	"time"
)

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

//var _ evaluate.Evaluator = LimitOverlayRule{}

func (r LimitOverlayRule) Evaluate(update poller.Update) (action.Action, error) {
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

	// FIXME: if autoAway switched off the heating, this rule will reset that after r.delay
	// workaround: don't remove the overlay if the heating is switched off (i.e. it's because of the autoAway rule)
	if state := tadotools.GetZoneState(update.ZoneInfo[r.zoneID]); state.Overlay == tado.PermanentOverlay && state.Heating() {
		a.State.(*State).mode = action.ZoneInAutoMode
		a.Delay = r.delay
		a.Reason = "manual temp setting detected"
	}

	r.logger.Debug("evaluated",
		slog.Bool("home", bool(update.Home)),
		slog.Any("result", a),
	)

	return a, nil
}
