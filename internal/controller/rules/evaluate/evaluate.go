package evaluate

import (
	"context"
	"github.com/clambin/tado-exporter/internal/poller"
	"time"
)

type TadoSetter interface {
	SetZoneOverlay(context.Context, int, float64) error
	DeleteZoneOverlay(context.Context, int) error
	SetHomeState(ctx context.Context, home bool) error
}

type Evaluator interface {
	Evaluate(update poller.Update) (Evaluation, error)
}

type Evaluation struct {
	Do    func(context.Context, TadoSetter) error
	Delay time.Duration
	// TODO: need to distinguish between actions that switch off & on heating (as off takes priority)?
	Reason string
}

func (e Evaluation) IsAction() bool {
	return e.Do != nil
}
