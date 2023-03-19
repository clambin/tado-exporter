package logger

import (
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/slack-go/slack"
)

type SlackLogger struct {
	Bot slackbot.SlackBot
}

var _ Logger = &SlackLogger{}

func (s SlackLogger) Log(action Action, state rules.TargetState) {
	_ = s.Bot.Send("", []slack.Attachment{{
		Color: "good",
		Title: state.ZoneName + ": " + state.Reason,
		Text:  buildMessage(action, state),
	}})
}
