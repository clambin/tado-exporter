package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"log/slog"
)

type SLogNotifier struct {
}

var _ Notifier = &SLogNotifier{}

func (s SLogNotifier) Notify(action Action, next rules.Action) {
	slog.Info(next.ZoneName+": "+buildMessage(action, next), "reason", next.Reason)
}