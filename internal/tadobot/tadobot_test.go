package tadobot

import (
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestDoVersion(t *testing.T) {
	bot := TadoBot{}

	responses := bot.doVersion()
	sort.Strings(responses)

	assert.Equal(t, "tado "+version.BuildVersion, responses[0])
}

func TestProcessMessage(t *testing.T) {
	bot, _ := Create("", nil)
	bot.userID = "12345678"

	var attachment *slack.Attachment

	attachment = bot.processMessage("Hello world")
	assert.Nil(t, attachment)
	attachment = bot.processMessage("<@12345678> Hello")
	assert.NotNil(t, attachment)
	assert.Equal(t, "Unknown command \"Hello\"", attachment.Title)
	attachment = bot.processMessage("<@12345678> version")
	assert.NotNil(t, attachment)
	assert.Equal(t, "tado "+version.BuildVersion, attachment.Text)
}

func TestProcessEvent(t *testing.T) {
	bot, _ := Create("", map[string]CallbackFunc{
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
			Text:    "<@987654321> hello",
		},
	}}

	_, attachments, stop = bot.processEvent(msg)
	assert.False(t, stop)
	if assert.Len(t, attachments, 1) {
		assert.Equal(t, "hello world", attachments[0].Text)
	}

	msg = slack.RTMEvent{Type: "invalid_auth", Data: &slack.InvalidAuthEvent{}}
	_, _, stop = bot.processEvent(msg)

	assert.True(t, stop)
}

func doHello() (responses []string) {
	responses = append(responses, "hello world")
	return
}
