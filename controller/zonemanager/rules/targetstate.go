package rules

import (
	"golang.org/x/exp/slog"
	"time"
)

var _ slog.LogValuer = &TargetState{}

type TargetState struct {
	ZoneID   int
	ZoneName string
	Action   bool
	State    ZoneState
	Delay    time.Duration
	Reason   string
}

func (s TargetState) LogValue() slog.Value {
	values := make([]slog.Attr, 3, 6)
	values[0] = slog.Int("id", s.ZoneID)
	values[1] = slog.String("name", s.ZoneName)
	values[2] = slog.Bool("action", s.Action)

	if s.Action {
		values = append(values, slog.Group("state", s.State.LogValue().Group()...))
		values = append(values, slog.Duration("delay", s.Delay))
	}
	values = append(values, slog.String("reason", s.Reason))
	return slog.GroupValue(values...)
}
