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
	values := []slog.Attr{
		slog.Bool("action", e.IsAction()),
		slog.String("reason", e.Reason),
	}
	if e.IsAction() {
		values = append(values,
			slog.Duration("delay", e.Delay),
			slog.Any("state", e.State),
		)
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

type Mode int

func (m Mode) String() string {
	if name, ok := modeNames[m]; ok {
		return name
	}
	return "unknown"
}

const (
	NoAction Mode = iota
	HomeInHomeMode
	HomeInAwayMode
	ZoneInOverlayMode
	ZoneInAutoMode
)

var modeNames = map[Mode]string{
	NoAction:          "no action",
	HomeInHomeMode:    "home",
	HomeInAwayMode:    "away",
	ZoneInOverlayMode: "overlay",
	ZoneInAutoMode:    "auto",
}
