package tadobot

import (
	"errors"
	"github.com/clambin/tado-exporter/internal/version"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"strings"
)

type CommandFunc func(args ...string) []slack.Attachment
type PostChannel chan []slack.Attachment

type TadoBot struct {
	PostChannel PostChannel

	slackClient *slack.Client
	slackRTM    *slack.RTM
	slackToken  string
	userID      string
	channelIDs  []string
	callbacks   map[string]CommandFunc
	reconnect   bool
}

// Create connects to a slackbot designated by token
func Create(slackToken string, callbacks map[string]CommandFunc) (bot *TadoBot, err error) {
	bot = &TadoBot{
		PostChannel: make(chan []slack.Attachment),
		slackClient: slack.New(slackToken),
		slackToken:  slackToken,
	}
	bot.callbacks = map[string]CommandFunc{
		"help":    bot.doHelp,
		"version": bot.doVersion,
	}
	if callbacks != nil {
		for name, callbackFunction := range callbacks {
			bot.callbacks[name] = callbackFunction
		}
	}
	bot.channelIDs, err = bot.getAllChannels()
	return
}

// Run the slackbot
func (bot *TadoBot) Run() (err error) {
	bot.slackRTM = bot.slackClient.NewRTM()
	go bot.slackRTM.ManageConnection()

loop:
	for {
		var (
			channel     string
			attachments []slack.Attachment
			stop        bool
		)

		select {
		case msg := <-bot.slackRTM.IncomingEvents:
			channel, attachments, stop = bot.processEvent(msg)
			if stop {
				break loop
			}
		case attachments = <-bot.PostChannel:
		}

		if len(attachments) > 0 {
			channels := bot.channelIDs
			if channel != "" {
				channels = []string{channel}
			}
			for _, channelID := range channels {
				if _, _, err = bot.slackRTM.PostMessage(
					channelID,
					slack.MsgOptionAttachments(attachments...),
					slack.MsgOptionAsUser(true),
				); err != nil {
					log.WithField("err", err).Warning("failed to send on slack")
				}
			}

			log.WithField("err", err).Debug("sent a message")
		}
	}
	return
}

func (bot *TadoBot) processEvent(msg slack.RTMEvent) (channel string, attachments []slack.Attachment, stop bool) {
	switch ev := msg.Data.(type) {
	// case *slack.HelloEvent:
	//	log.WithField("ev", ev).Debug("hello")
	case *slack.ConnectedEvent:
		bot.userID = ev.Info.User.ID
		if bot.reconnect == false {
			log.WithField("userID", bot.userID).Info("tadoBot connected to slack")
			bot.reconnect = true
		} else {
			log.Debug("tadoBot reconnected to slack")
		}
	case *slack.MessageEvent:
		log.WithFields(log.Fields{
			"name":     ev.Name,
			"user":     ev.User,
			"channel":  ev.Channel,
			"type":     ev.Type,
			"userName": ev.Username,
			"botID":    ev.BotID,
		}).Debug("message received: " + ev.Text)
		channel = ev.Channel
		attachments = bot.processMessage(ev.Text)
	case *slack.RTMError:
		log.WithField("error", ev.Error()).Error("error reading slack RTM connection")
	case *slack.InvalidAuthEvent:
		log.Error("invalid credentials")
		stop = true
	}
	return
}

func (bot *TadoBot) processMessage(text string) (attachments []slack.Attachment) {
	// check if we're mentioned
	log.WithField("text", text).Debug("processing slack chatter")
	command, args := bot.parseCommand(text)
	if command != "" {
		if callback, ok := bot.callbacks[command]; ok {
			attachments = callback(args...)
			log.WithFields(log.Fields{
				"command": command,
				"outputs": len(attachments),
			}).Debug("command run")
		} else {
			attachments = append(attachments, slack.Attachment{
				Color: "bad",
				Text:  "invalid command",
			})
		}
	}
	return
}

// getAllChannels returns all channels the bot can post on.
// This is either the bot's direct channel or any channels the bot has been invited to
func (bot *TadoBot) getAllChannels() (channelIDs []string, err error) {
	params := &slack.GetConversationsForUserParameters{
		Cursor: "",
		Limit:  0,
		Types: []string{
			"public_channel", "private_channel", "im",
		},
	}

	var channels []slack.Channel
	if channels, _, err = bot.slackClient.GetConversationsForUser(params); err == nil {

		for _, channel := range channels {
			log.WithFields(log.Fields{
				"name":      channel.Name,
				"id":        channel.ID,
				"isChannel": channel.IsChannel,
				"isPrivate": channel.IsPrivate,
				"isIM":      channel.IsIM,
			}).Debug("found a channel")

			channelIDs = append(channelIDs, channel.ID)
		}
	}

	log.WithFields(log.Fields{
		"channelIDs": channelIDs,
		"err":        err,
	}).Debug("found channels")

	return
}

func (bot *TadoBot) doHelp(_ ...string) []slack.Attachment {
	var commands = make([]string, 0)
	for command := range bot.callbacks {
		commands = append(commands, command)
	}
	return []slack.Attachment{
		{
			Color: "good",
			Title: "supported commands",
			Text:  strings.Join(commands, ", "),
		},
	}
}

func (bot *TadoBot) doVersion(_ ...string) []slack.Attachment {
	return []slack.Attachment{
		{
			Color: "good",
			Text:  "tado " + version.BuildVersion,
		},
	}
}

func SendMessage(postChannel PostChannel, color, title, text string) (err error) {
	if postChannel != nil {
		postChannel <- []slack.Attachment{
			{
				Color: color,
				Title: title,
				Text:  text,
			},
		}
	} else {
		err = errors.New("invalid channel supplied")
	}

	log.WithField("err", err).Debug("sent a message")

	return
}
