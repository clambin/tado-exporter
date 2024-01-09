package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/slack-go/slack"
)

type SlackNotifier struct {
	Slack SlackSender
}

type SlackSender interface {
	Send(channel string, attachments []slack.Attachment) error
}

var _ Notifier = &SlackNotifier{}

func (s SlackNotifier) Notify(action ScheduleType, state action.Action) {
	_ = s.Slack.Send("", []slack.Attachment{{
		Color: "good",
		Title: buildMessage(action, state),
		Text:  state.Reason,
	}})
}
