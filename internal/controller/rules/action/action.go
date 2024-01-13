package action

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

var _ slog.LogValuer = Action{}

type Action struct {
	Delay  time.Duration
	Reason string
	Label  string
	State  State
}

type TadoSetter interface {
	SetZoneOverlay(context.Context, int, float64) error
	DeleteZoneOverlay(context.Context, int) error
	SetHomeState(ctx context.Context, home bool) error
}

type State interface {
	slog.LogValuer
	fmt.Stringer
	Do(context.Context, TadoSetter) error
	IsEqual(State) bool
	Mode() Mode
}

func (e Action) LogValue() slog.Value {
	values := make([]slog.Attr, 2, 5)
	values[0] = slog.Bool("action", e.IsAction())
	values[1] = slog.String("reason", e.Reason)

	if e.Label != "" {
		values = append(values, slog.String("label", e.Label))
	}
	if e.IsAction() {
		values = append(values, slog.Duration("delay", e.Delay), slog.Any("state", e.State))
	}
	return slog.GroupValue(values...)
}

func (e Action) IsAction() bool {
	return e.State != nil && e.State.Mode() != NoAction
}

func (e Action) String() string {
	if e.State == nil {
		return "no action"
	}
	return e.State.String()
}
