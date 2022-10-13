package logger

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"time"
)

type Logger interface {
	Log(action Action, state *rules.NextState)
}

type Action int

const (
	Queued Action = iota
	Done
	Canceled
)

type Loggers []Logger

func (ls Loggers) Log(action Action, state *rules.NextState) {
	for _, l := range ls {
		l.Log(action, state)
	}
}

func buildMessage(action Action, state *rules.NextState) string {
	switch action {
	case Queued:
		return getAction(state) + " in " + state.Delay.Round(time.Second).String()
	case Done:
		return getAction(state)
	case Canceled:
		return "cancel " + getAction(state)
	}
	return ""
}

func getAction(state *rules.NextState) (text string) {
	switch state.State {
	case tado.ZoneStateAuto:
		text = "moving to auto mode"
	case tado.ZoneStateOff:
		text = "switching off heating"
	}

	return
}

func getReason(action Action, state *rules.NextState) string {
	if action == Canceled {
		return state.CancelReason
	}
	return state.ActionReason
}
