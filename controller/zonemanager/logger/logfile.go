package logger

import (
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	log "github.com/sirupsen/logrus"
)

type StdOutLogger struct {
}

var _ Logger = &StdOutLogger{}

func (l *StdOutLogger) Log(action Action, next *rules.NextState) {
	log.WithField("reason", getReason(action, next)).Info(next.ZoneName + ": " + buildMessage(action, next))
}
