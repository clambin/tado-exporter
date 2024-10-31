package action

import (
	"context"
	"fmt"
	"github.com/clambin/tado/v2"
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

type TadoClient interface {
	SetPresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, body tado.SetPresenceLockJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error)
	SetZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, body tado.SetZoneOverlayJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error)
	DeleteZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, reqEditors ...tado.RequestEditorFn) (*tado.DeleteZoneOverlayResponse, error)
}

type State interface {
	slog.LogValuer
	fmt.Stringer
	Do(context.Context, TadoClient) error
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

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Mode int

func (m Mode) String() string {
	if m >= 0 && int(m) < len(modeNames) {
		return modeNames[m]
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

var modeNames = []string{
	"no action",
	"home",
	"away",
	"overlay",
	"auto",
}
