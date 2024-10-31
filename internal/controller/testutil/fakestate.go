package testutil

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"log/slog"
)

var _ action.State = FakeState{}

type FakeState struct{ ModeValue action.Mode }

func (f FakeState) LogValue() slog.Value {
	return slog.GroupValue(slog.String("mode", f.ModeValue.String()))
}

func (f FakeState) String() string {
	return f.ModeValue.String()
}

func (f FakeState) Do(_ context.Context, _ action.TadoClient) error {
	return nil
}

func (f FakeState) IsEqual(state action.State) bool {
	return f.ModeValue == state.Mode()
}

func (f FakeState) Mode() action.Mode {
	return f.ModeValue
}
