package slackbot

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type SlackClient interface {
	Run(ctx context.Context)
	Send(message SlackMessage) (err error)
	GetChannels() (channelIDs []string, err error)
}

type slackClient struct {
	NextEvent chan slack.RTMEvent

	slackClient *slack.Client
	slackRTM    *slack.RTM
	channels    []string
}

type SlackMessage struct {
	Channel     string
	Attachments []slack.Attachment
}

func newClient(token string, events chan slack.RTMEvent) (client SlackClient) {
	return &slackClient{
		NextEvent:   events,
		slackClient: slack.New(token),
	}
}

func (client *slackClient) Run(ctx context.Context) {
	client.slackRTM = client.slackClient.NewRTM()
	go client.slackRTM.ManageConnection()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case msg := <-client.slackRTM.IncomingEvents:
			client.NextEvent <- msg
		}
	}
}

// GetChannels returns all channels the bot can post on.
// This is either the bot's direct channel or any getChannels the bot has been invited to
func (client *slackClient) GetChannels() (channelIDs []string, err error) {
	var channels []slack.Channel
	channels, _, err = client.slackClient.GetConversationsForUser(&slack.GetConversationsForUserParameters{
		Types: []string{"public_channel", "private_channel", "im"},
	})
	if err != nil {
		return nil, err
	}

	for _, channel := range channels {
		channelIDs = append(channelIDs, channel.ID)
	}

	log.WithError(err).WithField("channelIDs", channelIDs).Debug("found getChannels")
	return
}

// Send a message to slack.  if no channel is specified, the message is broadcast to all getChannels
func (client *slackClient) Send(message SlackMessage) (err error) {
	_, _, err = client.slackRTM.PostMessage(
		message.Channel,
		slack.MsgOptionAttachments(message.Attachments...),
		slack.MsgOptionAsUser(true),
	)

	log.WithError(err).Debug("sent a message")
	return
}
