package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"time"
)

type Notifier interface {
	Notify(action Action, state rules.Action)
}

type Action int

const (
	Queued Action = iota
	Done
	Canceled
)

type Notifiers []Notifier

func (n Notifiers) Notify(action Action, state rules.Action) {
	for _, l := range n {
		l.Notify(action, state)
	}
}

func buildMessage(action Action, state rules.Action) string {
	a := state.State.String()
	switch action {
	case Queued:
		return a + " in " + state.Delay.Round(time.Second).String()
	case Done:
		return a
	case Canceled:
		return "canceling " + a
	}
	return ""
}
