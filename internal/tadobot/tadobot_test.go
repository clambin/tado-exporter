package tadobot

import (
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestDoVersion(t *testing.T) {
	bot := TadoBot{}

	responses := bot.doVersion()
	if assert.Len(t, responses, 1) {
		assert.Equal(t, "tado "+version.BuildVersion, responses[0].Text)
	}
}

func TestProcessMessage(t *testing.T) {
	bot, _ := Create("", nil)
	bot.userID = "12345678"

	var attachments []slack.Attachment

	attachments = bot.processMessage("Hello world")
	assert.Len(t, attachments, 0)
	attachments = bot.processMessage("<@12345678> Hello")
	assert.Len(t, attachments, 1)
	assert.Equal(t, "invalid command", attachments[0].Text)
	attachments = bot.processMessage("<@12345678> version")
	assert.Len(t, attachments, 1)
	assert.Equal(t, "tado "+version.BuildVersion, attachments[0].Text)
}

func TestProcessEvent(t *testing.T) {
	bot, _ := Create("", map[string]CommandFunc{
		"hello": doHello,
	})

	msg := slack.RTMEvent{Type: "connected", Data: &slack.ConnectedEvent{
		ConnectionCount: 1,
		Info: &slack.Info{
			User: &slack.UserDetails{ID: "123456789"},
		},
	}}

	var (
		attachments []slack.Attachment
		stop        bool
	)
	_, _, stop = bot.processEvent(msg)
	assert.False(t, stop)
	assert.Equal(t, "123456789", bot.userID)
	assert.True(t, bot.reconnect)

	msg = slack.RTMEvent{Type: "connected", Data: &slack.ConnectedEvent{
		ConnectionCount: 1,
		Info: &slack.Info{
			User: &slack.UserDetails{ID: "987654321"},
		},
	}}

	_, _, stop = bot.processEvent(msg)
	assert.False(t, stop)
	assert.Equal(t, "987654321", bot.userID)
	assert.True(t, bot.reconnect)

	msg = slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "some_channel",
			Text:    "some text",
		},
	}}

	_, attachments, stop = bot.processEvent(msg)
	assert.False(t, stop)
	assert.Len(t, attachments, 0)

	msg = slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "some_channel",
			Text:    "<@987654321> version",
		},
	}}

	_, attachments, stop = bot.processEvent(msg)
	assert.False(t, stop)
	if assert.Len(t, attachments, 1) {
		assert.Equal(t, "tado "+version.BuildVersion, attachments[0].Text)
	}

	msg = slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "some_channel",
			Text:    "<@987654321> hello how are you",
		},
	}}

	_, attachments, stop = bot.processEvent(msg)
	assert.False(t, stop)
	if assert.Len(t, attachments, 1) {
		assert.Equal(t, "hello world how, are, you", attachments[0].Text)
	}

	msg = slack.RTMEvent{Type: "invalid_auth", Data: &slack.InvalidAuthEvent{}}
	_, _, stop = bot.processEvent(msg)

	assert.True(t, stop)
}

func doHello(args ...string) (responses []slack.Attachment) {
	responses = []slack.Attachment{
		{
			Text: "hello world " + strings.Join(args, ", "),
		},
	}
	return
}
