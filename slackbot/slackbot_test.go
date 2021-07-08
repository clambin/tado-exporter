package slackbot_test

import (
	"context"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/version"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSlackBot_InvalidAuth(t *testing.T) {
	client, err := slackbot.Create("test", "12345678", nil)
	assert.NoError(t, err)

	events := make(chan slack.RTMEvent)
	output := make(chan slackbot.SlackMessage)
	server := &mockSlack{
		userID:    "1234",
		eventsIn:  events,
		eventsOut: client.Events,
		output:    output,
	}
	client.SlackClient = server

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err = client.Run(ctx)
		assert.NoError(t, err)
	}()

	events <- server.InvalidAuthEvent()

	time.Sleep(time.Second)
	assert.Panics(t, func() { close(client.PostChannel) })
}

func TestSlackBot_Commands(t *testing.T) {
	callbacks := map[string]slackbot.CommandFunc{
		"test": func(_ ...string) []slack.Attachment {
			return []slack.Attachment{
				{
					Text: "hello world!",
				},
			}
		},
	}
	client, err := slackbot.Create("test-client-"+version.BuildVersion, "12345678", callbacks)
	assert.NoError(t, err)

	events := make(chan slack.RTMEvent)
	output := make(chan slackbot.SlackMessage)
	server := &mockSlack{
		userID:    "1234",
		channels:  []string{"1", "2", "3"},
		eventsIn:  events,
		eventsOut: client.Events,
		output:    output,
	}
	client.SlackClient = server

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err = client.Run(ctx)
		assert.NoError(t, err)
	}()

	events <- server.ConnectedEvent()

	events <- server.MessageEvent("1", "random chatter. should be ignored")

	events <- server.MessageEvent("1", "<@1234> version")
	response := <-output
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "test-client-"+version.BuildVersion, response.Attachments[0].Text)
	}

	events <- server.MessageEvent("2", "<@1234> help")
	response = <-output
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "help, test, version", response.Attachments[0].Text)
	}

	events <- server.MessageEvent("3", "<@1234> test")
	response = <-output
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "hello world!", response.Attachments[0].Text)
	}

	events <- server.MessageEvent("3", "<@1234> notacommand")
	response = <-output
	if assert.Len(t, response.Attachments, 1) {
		assert.Equal(t, "", response.Attachments[0].Title)
		assert.Equal(t, "invalid command", response.Attachments[0].Text)
		assert.Equal(t, "bad", response.Attachments[0].Color)
	}

}

func TestSlackBot_Post(t *testing.T) {
	client, err := slackbot.Create("test-client"+version.BuildVersion, "12345678", nil)
	assert.NoError(t, err)

	events := make(chan slack.RTMEvent)
	output := make(chan slackbot.SlackMessage)
	server := &mockSlack{
		userID:    "1234",
		channels:  []string{"1", "2", "3"},
		eventsIn:  events,
		eventsOut: client.Events,
		output:    output,
	}
	client.SlackClient = server

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err = client.Run(ctx)
		assert.NoError(t, err)
	}()

	events <- server.ConnectedEvent()
	client.PostChannel <- []slack.Attachment{{
		Text: "Hello world!",
	}}

	for i := 0; i < len(server.channels); i++ {
		msg := <-output

		if assert.Len(t, msg.Attachments, 1) {
			assert.Equal(t, "Hello world!", msg.Attachments[0].Text)
		}
	}
}

type mockSlack struct {
	userID    string
	channels  []string
	eventsIn  chan slack.RTMEvent
	eventsOut chan slack.RTMEvent
	output    chan slackbot.SlackMessage
}

func (m mockSlack) Run(ctx context.Context) {
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case msg := <-m.eventsIn:
			m.eventsOut <- msg
		}
	}
}

func (m mockSlack) Send(message slackbot.SlackMessage) (err error) {
	m.output <- message
	return nil
}

func (m mockSlack) GetChannels() (channelIDs []string, err error) {
	return m.channels, nil
}

func (m mockSlack) ConnectedEvent() slack.RTMEvent {
	return slack.RTMEvent{Type: "connected", Data: &slack.ConnectedEvent{
		ConnectionCount: 1,
		Info: &slack.Info{
			User: &slack.UserDetails{
				ID: m.userID,
			},
		},
	}}
}

func (m mockSlack) InvalidAuthEvent() slack.RTMEvent {
	return slack.RTMEvent{Type: "invalid auth", Data: &slack.InvalidAuthEvent{}}
}

func (m mockSlack) MessageEvent(channel string, message string) slack.RTMEvent {
	return slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{Msg: slack.Msg{
		Channel: channel,
		Text:    message,
	}}}
}
