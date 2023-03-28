package notifier

import (
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/slack-go/slack"
)

type SlackNotifier struct {
	Bot slackbot.SlackBot
}

var _ Notifier = &SlackNotifier{}

func (s SlackNotifier) Notify(action Action, state rules.TargetState) {
	_ = s.Bot.Send("", []slack.Attachment{{
		Color: "good",
		Title: state.ZoneName + ": " + buildMessage(action, state),
		Text:  state.Reason,
	}})
}
