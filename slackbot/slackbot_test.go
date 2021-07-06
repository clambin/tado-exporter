package slackbot

import (
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProcessMessage(t *testing.T) {
	bot, _ := Create("testBot", "", map[string]CommandFunc{})
	bot.userID = "12345678"

	var attachments []slack.Attachment

	attachments = bot.processMessage("Hello world")
	assert.Len(t, attachments, 0)
	attachments = bot.processMessage("<@12345678> Hello")
	assert.Len(t, attachments, 1)
	assert.Equal(t, "invalid command", attachments[0].Text)
	attachments = bot.processMessage("<@12345678> version")
	assert.Len(t, attachments, 1)
	assert.Equal(t, "testBot", attachments[0].Text)
}

func TestProcessEvent(t *testing.T) {
	bot, _ := Create("testBot", "", nil)
	bot.RegisterCallback("hello", doHello)

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
		assert.Equal(t, "testBot", attachments[0].Text)
	}

	msg = slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{
		Msg: slack.Msg{
			Channel: "some_channel",
			Text:    "<@987654321> help",
		},
	}}

	_, attachments, stop = bot.processEvent(msg)
	assert.False(t, stop)
	if assert.Len(t, attachments, 1) {
		assert.Equal(t, "hello, help, version", attachments[0].Text)
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

func TestSlackBot_Run(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	bot, err := Create("testBot", "", nil)
	if assert.Nil(t, err) {
		go func() {
			_ = bot.Run()
		}()

		// we're connected to slack
		bot.events <- slack.RTMEvent{Type: "connected", Data: &slack.ConnectedEvent{
			ConnectionCount: 1,
			Info: &slack.Info{
				User: &slack.UserDetails{ID: "987654321"},
			},
		}}
		// some non-actionable chatter
		bot.events <- slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "some_channel",
				Text:    "some text",
			},
		}}
		// ask for version
		bot.events <- slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{
			Msg: slack.Msg{
				Channel: "some_channel",
				Text:    "<@987654321> version",
			},
		}}
		response := <-bot.messages

		assert.Equal(t, "some_channel", response.Channel)
		if assert.Len(t, response.Attachments, 1) {
			assert.Equal(t, "testBot", response.Attachments[0].Text)
		}

		// check posting works
		bot.PostChannel <- []slack.Attachment{
			{
				Color: "good",
				Text:  "hello world",
			},
		}
		response = <-bot.messages

		if assert.Len(t, response.Attachments, 1) {
			assert.Equal(t, "hello world", response.Attachments[0].Text)
		}

		// check that the bot will stop
		bot.events <- slack.RTMEvent{Type: "invalid_auth", Data: &slack.InvalidAuthEvent{}}

		assert.Eventually(t, func() bool {
			_, ok := <-bot.messages
			return !ok
		}, 1*time.Second, 10*time.Millisecond)
	}
}

func TestEndToEnd(t *testing.T) {
	if token := os.Getenv("SLACK_TOKEN"); token != "" {
		bot, err := Create("testBot", token, nil)

		if assert.Nil(t, err) {
			_ = bot.Run()
		}
	}
}
