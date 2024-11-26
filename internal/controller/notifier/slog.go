package notifier

import (
	"log/slog"
)

type SLogNotifier struct {
	Logger *slog.Logger
}

var _ Notifier = &SLogNotifier{}

func (s SLogNotifier) Notify(msg string) {
	s.Logger.Info(msg)
}
