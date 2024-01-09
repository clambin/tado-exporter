package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"time"
)

type Notifier interface {
	Notify(state ScheduleType, action action.Action)
}

type ScheduleType int

const (
	Queued ScheduleType = iota
	Done
	Canceled
)

type Notifiers []Notifier

func (n Notifiers) Notify(state ScheduleType, action action.Action) {
	for _, l := range n {
		l.Notify(state, action)
	}
}

func buildMessage(state ScheduleType, action action.Action) string {
	var prefix string
	if action.Label != "" {
		prefix = action.Label + ": "
	}
	a := action.String()
	switch state {
	case Queued:
		return prefix + a + " in " + action.Delay.Round(time.Second).String()
	case Done:
		return prefix + a
	case Canceled:
		return prefix + "canceling " + a
	default:
		return ""
	}
}
