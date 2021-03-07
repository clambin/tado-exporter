package slackbot

import (
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// SlackClient abstracts the interface to/from slack so it can be stubbed in unit tests
type SlackClient struct {
	NextEvent chan slack.RTMEvent
	Messages  chan Message

	slackClient *slack.Client
	slackRTM    *slack.RTM
	channels    []string
}

// Message contains a message to be sent to slack
type Message struct {
	Channel     string
	Attachments []slack.Attachment
}

// NewClient creates a new SlackClient
func NewClient(token string, events chan slack.RTMEvent, messages chan Message) (client *SlackClient) {
	client = &SlackClient{
		NextEvent:   events,
		Messages:    messages,
		slackClient: slack.New(token),
	}
	var err error
	if client.channels, err = client.getAllChannels(); err != nil {
		client = nil
	}
	return
}

func (client *SlackClient) Run() {
	client.slackRTM = client.slackClient.NewRTM()
	go client.slackRTM.ManageConnection()

	for {
		select {
		case msg := <-client.slackRTM.IncomingEvents:
			client.NextEvent <- msg
		case msg := <-client.Messages:
			_ = client.send(msg)
		}
	}
}

// getAllChannels returns all channels the bot can post on.
// This is either the bot's direct channel or any channels the bot has been invited to
func (client *SlackClient) getAllChannels() (channelIDs []string, err error) {
	params := &slack.GetConversationsForUserParameters{
		Types: []string{"public_channel", "private_channel", "im"},
	}
	var channels []slack.Channel
	if channels, _, err = client.slackClient.GetConversationsForUser(params); err == nil {

		for _, channel := range channels {
			channelIDs = append(channelIDs, channel.ID)
		}
	}
	log.WithFields(log.Fields{
		"channelIDs": channelIDs,
		"err":        err,
	}).Debug("found channels")
	return
}

// send a message to slack.  if no channel is specified, the message is broadcast to all channels
func (client *SlackClient) send(message Message) (err error) {
	channels := client.channels
	if message.Channel != "" {
		channels = []string{message.Channel}
	}
	for _, channelID := range channels {
		if _, _, err = client.slackRTM.PostMessage(
			channelID,
			slack.MsgOptionAttachments(message.Attachments...),
			slack.MsgOptionAsUser(true),
		); err != nil {
			log.WithField("err", err).Warning("failed to send on slack")
		}
	}
	log.WithField("err", err).Debug("sent a message")
	return
}
