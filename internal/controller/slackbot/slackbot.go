package slackbot

import (
	"context"
	"github.com/clambin/go-common/slackbot"
	"github.com/slack-go/slack"
)

type SlackBot interface {
	Register(name string, command slackbot.CommandFunc)
	Run(ctx context.Context) error
	Send(channel string, attachments []slack.Attachment) error
}