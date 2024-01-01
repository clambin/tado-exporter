package zone

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/rules/evaluate"
	"github.com/clambin/tado-exporter/internal/poller"
	"time"
)

type LimitOverlayRule struct {
	zoneID   int
	zoneName string
	delay    time.Duration
}

func LoadLimitOverlay(id int, name string, cfg configuration.LimitOverlayConfiguration, _ poller.Update) (LimitOverlayRule, error) {
	return LimitOverlayRule{
		zoneID:   id,
		zoneName: name,
		delay:    cfg.Delay,
	}, nil
}

var _ evaluate.Evaluator = LimitOverlayRule{}

func (l LimitOverlayRule) Evaluate(update poller.Update) (evaluate.Evaluation, error) {
	var e evaluate.Evaluation

	e.Reason = "no manual temp setting detected"
	if state := GetZoneState(update.ZoneInfo[l.zoneID]); state.Overlay == tado.PermanentOverlay && state.Heating() {
		e.Do = func(ctx context.Context, setter evaluate.TadoSetter) error {
			return setter.DeleteZoneOverlay(ctx, l.zoneID)
		}
		e.Delay = l.delay
		e.Reason = "manual temp setting detected"
	}
	return e, nil
}
