package logger

import (
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/tado"
	"time"
)

type Logger interface {
	Log(action Action, state rules.TargetState)
}

type Action int

const (
	Queued Action = iota
	Done
	Canceled
)

type Loggers []Logger

func (ls Loggers) Log(action Action, state rules.TargetState) {
	for _, l := range ls {
		l.Log(action, state)
	}
}

func buildMessage(action Action, state rules.TargetState) string {
	switch action {
	case Queued:
		return getAction(state) + " in " + state.Delay.Round(time.Second).String()
	case Done:
		return getAction(state)
	case Canceled:
		return "canceling " + getAction(state)
	}
	return ""
}

func getAction(state rules.TargetState) string {
	switch state.State {
	case tado.ZoneStateAuto:
		return "moving to auto mode"
	case tado.ZoneStateOff:
		return "switching off heating"
	}
	return "unknown"
}
