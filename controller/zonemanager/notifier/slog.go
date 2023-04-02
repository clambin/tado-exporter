package notifier

import (
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"golang.org/x/exp/slog"
)

type SLogNotifier struct {
}

var _ Notifier = &SLogNotifier{}

func (s SLogNotifier) Notify(action Action, next rules.Action) {
	slog.Info(next.ZoneName+": "+buildMessage(action, next), "reason", next.Reason)
}
