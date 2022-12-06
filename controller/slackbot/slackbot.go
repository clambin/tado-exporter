package slackbot

import (
	"context"
	"github.com/clambin/go-common/slackbot"
	"github.com/slack-go/slack"
)

// SlackBot interface mimics github.com/go-tools/slackbot
//
//go:generate mockery --name SlackBot
type SlackBot interface {
	Register(name string, command slackbot.CommandFunc)
	Run(ctx context.Context) (err error)
	Send(channel string, attachments []slack.Attachment) error
}
