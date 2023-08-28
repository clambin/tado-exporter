package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/slackbot"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/slack-go/slack"
)

type SlackNotifier struct {
	Bot slackbot.SlackBot
}

var _ Notifier = &SlackNotifier{}

func (s SlackNotifier) Notify(action Action, state rules.Action) {
	_ = s.Bot.Send("", []slack.Attachment{{
		Color: "good",
		Title: state.ZoneName + ": " + buildMessage(action, state),
		Text:  state.Reason,
	}})
}
