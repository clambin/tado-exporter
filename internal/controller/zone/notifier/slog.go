package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"log/slog"
)

type SLogNotifier struct {
	Logger *slog.Logger
}

var _ Notifier = &SLogNotifier{}

func (s SLogNotifier) Notify(action Action, next rules.Action) {
	s.Logger.Info(next.ZoneName+": "+buildMessage(action, next), "reason", next.Reason)
}
