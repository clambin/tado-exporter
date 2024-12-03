package controller

import (
	"context"
	"log/slog"
	"time"
)

type action interface {
	State() state
	Delay() time.Duration
	Reason() string
	setReason(string)
	Description(includeDelay bool) string
	Do(context.Context, TadoClient, *slog.Logger) error
	slog.LogValuer
}

type coreAction struct {
	state
	reason string
	delay  time.Duration
}

func (a *coreAction) Delay() time.Duration {
	return a.delay
}

func (a *coreAction) Reason() string {
	return a.reason
}
func (a *coreAction) setReason(reason string) {
	a.reason = reason
}
func (a *coreAction) Description(includeDelay bool) string {
	text := a.state.String()
	if includeDelay {
		text += " in " + a.delay.String()
	}
	return text
}

func (a *coreAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("state", a.state.LogValue()),
		slog.Duration("delay", a.delay),
		slog.String("reason", a.reason),
	)
}
