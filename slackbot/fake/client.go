package fake

import (
	"context"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/slack-go/slack"
)

type Client struct {
	UserID    string
	Channels  []string
	EventsIn  chan slack.RTMEvent
	EventsOut chan slack.RTMEvent
	Output    chan slackbot.SlackMessage
}

func (client Client) Run(ctx context.Context) {
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case msg := <-client.EventsIn:
			client.EventsOut <- msg
		}
	}
}

func (client Client) Send(message slackbot.SlackMessage) (err error) {
	client.Output <- message
	return nil
}

func (client Client) GetChannels() (channelIDs []string, err error) {
	return client.Channels, nil
}

func (client Client) ConnectedEvent() slack.RTMEvent {
	return slack.RTMEvent{Type: "connected", Data: &slack.ConnectedEvent{
		ConnectionCount: 1,
		Info: &slack.Info{
			User: &slack.UserDetails{
				ID: client.UserID,
			},
		},
	}}
}

func (client Client) InvalidAuthEvent() slack.RTMEvent {
	return slack.RTMEvent{Type: "invalid_auth", Data: &slack.InvalidAuthEvent{}}
}

func (client Client) MessageEvent(channel string, message string) slack.RTMEvent {
	return slack.RTMEvent{Type: "message", Data: &slack.MessageEvent{Msg: slack.Msg{
		Channel: channel,
		Text:    message,
	}}}
}

func (client Client) RTMErrorEvent() slack.RTMEvent {
	return slack.RTMEvent{Type: "error", Data: &slack.RTMError{
		Code: 1,
		Msg:  "test. this will be ignored",
	}}
}
