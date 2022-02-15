package slackbot_test

import (
	"context"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/slackbot/fake"
	"github.com/clambin/tado-exporter/version"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSlackBot_Commands(t *testing.T) {
	callbacks := map[string]slackbot.CommandFunc{
		"test": func(_ context.Context, _ ...string) []slack.Attachment {
			return []slack.Attachment{
				{
					Text: "hello world!",
				},
			}
		},
	}
	client := slackbot.Create("test-client-"+version.BuildVersion, "12345678", callbacks)

	events := make(chan slack.RTMEvent)
	output := make(chan slackbot.SlackMessage)
	server := &fake.Client{
		UserID:    "1234",
		Channels:  []string{"1", "2", "3"},
		EventsIn:  events,
		EventsOut: client.Events,
		Output:    output,
	}
	client.SlackClient = server

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctx context.Context) {
		err := client.Run(ctx)
		assert.NoError(t, err)
	}(ctx)

	events <- server.ConnectedEvent()

	events <- server.MessageEvent("1", "random chatter. should be ignored")

	events <- server.MessageEvent("1", "<@1234> version")
	response := <-output
	assert.Equal(t, "1", response.Channel)
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "test-client-"+version.BuildVersion, response.Attachments[0].Text)
	}

	events <- server.MessageEvent("2", "<@1234> help")
	response = <-output
	assert.Equal(t, "2", response.Channel)
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "help, test, version", response.Attachments[0].Text)
	}

	events <- server.MessageEvent("3", "<@1234> test")
	response = <-output
	assert.Equal(t, "3", response.Channel)
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "hello world!", response.Attachments[0].Text)
	}

	events <- server.MessageEvent("3", "<@1234> notacommand")
	response = <-output
	assert.Equal(t, "3", response.Channel)
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "", response.Attachments[0].Title)
		assert.Equal(t, "invalid command", response.Attachments[0].Text)
		assert.Equal(t, "bad", response.Attachments[0].Color)
	}

}

func TestSlackBot_Post(t *testing.T) {
	client := slackbot.Create("test-client"+version.BuildVersion, "12345678", nil)

	events := make(chan slack.RTMEvent)
	output := make(chan slackbot.SlackMessage, 5)
	server := &fake.Client{
		UserID:    "1234",
		Channels:  []string{"1", "2", "3"},
		EventsIn:  events,
		EventsOut: client.Events,
		Output:    output,
	}
	client.SlackClient = server

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctx context.Context) {
		err := client.Run(ctx)
		assert.NoError(t, err)
	}(ctx)

	events <- server.ConnectedEvent()
	events <- server.InvalidAuthEvent()
	events <- server.RTMErrorEvent()
	events <- server.ConnectedEvent()
	_ = client.Send("", "", "", "Hello world!")

	for i := 0; i < len(server.Channels); i++ {
		msg := <-output

		assert.Contains(t, server.Channels, msg.Channel)
		if assert.Len(t, msg.Attachments, 1) {
			assert.Equal(t, "Hello world!", msg.Attachments[0].Text)
		}
	}

	client.GetPostChannel() <- []slack.Attachment{
		{
			Color: "good",
			Title: "test",
			Text:  "hello world",
		},
	}

	for i := 0; i < len(server.Channels); i++ {
		msg := <-output

		assert.Contains(t, server.Channels, msg.Channel)
		require.Len(t, msg.Attachments, 1)
		assert.Equal(t, "good", msg.Attachments[0].Color)
		assert.Equal(t, "test", msg.Attachments[0].Title)
		assert.Equal(t, "hello world", msg.Attachments[0].Text)
	}
}
