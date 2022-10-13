package logger

import (
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/slack-go/slack"
)

type SlackLogger struct {
	slackbot.PostChannel
}

var _ Logger = &SlackLogger{}

func (s SlackLogger) Log(action Action, state *rules.NextState) {
	s.PostChannel <- []slack.Attachment{{
		Color: "good",
		Title: state.ZoneName + ": " + getReason(action, state),
		Text:  buildMessage(action, state),
	}}
}
