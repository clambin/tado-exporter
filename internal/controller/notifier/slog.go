package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"log/slog"
)

type SLogNotifier struct {
	Logger *slog.Logger
}

var _ Notifier = &SLogNotifier{}

func (s SLogNotifier) Notify(state ScheduleType, action action.Action) {
	s.Logger.Info(buildMessage(state, action), "reason", action.Reason)
}
