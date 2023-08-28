package rules

import (
	"log/slog"
	"time"
)

var _ slog.LogValuer = &Action{}

type Action struct {
	ZoneID   int
	ZoneName string
	Action   bool
	Reason   string
	State    ZoneState
	Delay    time.Duration
}

func (s Action) LogValue() slog.Value {
	values := make([]slog.Attr, 3, 6)
	values[0] = slog.Int("id", s.ZoneID)
	values[1] = slog.String("name", s.ZoneName)
	values[2] = slog.Bool("action", s.Action)

	if s.Action {
		values = append(values, slog.Any("state", s.State.LogValue().Any()))
		values = append(values, slog.Duration("delay", s.Delay))
	}
	values = append(values, slog.String("reason", s.Reason))
	return slog.GroupValue(values...)
}
