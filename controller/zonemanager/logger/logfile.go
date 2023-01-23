package logger

import (
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"golang.org/x/exp/slog"
)

type StdOutLogger struct {
}

var _ Logger = &StdOutLogger{}

func (l *StdOutLogger) Log(action Action, next rules.NextState) {
	slog.Info(next.ZoneName+": "+buildMessage(action, next), "reason", getReason(action, next))
}
