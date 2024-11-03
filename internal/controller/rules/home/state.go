package home

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado/v2"
	"log/slog"
)

var _ action.State = State{}

type State struct {
	mode   action.Mode
	homeId tado.HomeId
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

func (s State) Do(ctx context.Context, setter action.TadoClient) error {
	var homePresence tado.HomePresence
	switch s.mode {
	case action.HomeInHomeMode:
		homePresence = tado.HOME
	case action.HomeInAwayMode:
		homePresence = tado.AWAY
	default:
		return fmt.Errorf("invalid home mode: %s", s.mode.String())
	}

	_, err := setter.SetPresenceLockWithResponse(ctx, s.homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &homePresence})
	return err
}

func (s State) IsEqual(state action.State) bool {
	return s.mode == state.Mode()
}

func (s State) Mode() action.Mode {
	return s.mode
}
