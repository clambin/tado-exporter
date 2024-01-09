package rules

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"log/slog"
)

var _ action.State = State{}

type State struct {
	mode action.Mode
}

func (s State) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", "home"),
		slog.String("mode", s.mode.String()),
	)
}

func (s State) String() string {
	return "setting home to " + s.mode.String() + " mode"
}

func (s State) Do(ctx context.Context, setter action.TadoSetter) error {
	return setter.SetHomeState(ctx, s.mode == action.HomeInHomeMode)
}

func (s State) IsEqual(state action.State) bool {
	return s.mode == state.Mode()
}

func (s State) Mode() action.Mode {
	return s.mode
}
