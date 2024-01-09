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
		logger:   logger.With("rule", "limitOverlay"),
	}, nil
}

//var _ evaluate.Evaluator = LimitOverlayRule{}

func (r LimitOverlayRule) Evaluate(update poller.Update) (action.Action, error) {
	s := State{
		zoneID:   r.zoneID,
		zoneName: r.zoneName,
		mode:     action.NoAction,
	}
	e := action.Action{Label: r.zoneName, Reason: "no manual temp setting detected"}

	if !update.Home {
		e.State = s
		e.Reason = "home in AWAY mode"
		return e, nil
	}

	if state := tadotools.GetZoneState(update.ZoneInfo[r.zoneID]); state.Overlay == tado.PermanentOverlay {
		s.mode = action.ZoneInAutoMode
		e.Delay = r.delay
		e.Reason = "manual temp setting detected"
	}
	e.State = s

	r.logger.Debug("evaluated",
		slog.Bool("home", bool(update.Home)),
		slog.Any("result", e),
	)

	return e, nil
}
