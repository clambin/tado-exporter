package tadobot

import (
	"github.com/clambin/tado-exporter/internal/version"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"strings"
)

type CallbackFunc func() []string

type TadoBot struct {
	slackClient *slack.Client
	slackRTM    *slack.RTM
	slackToken  string
	userID      string
	channelIDs  []string
	callbacks   map[string]CallbackFunc
	reconnect   bool
}

// Create connects to a slackbot designated by token
func Create(slackToken string, callbacks map[string]CallbackFunc) (bot *TadoBot, err error) {
	bot = &TadoBot{
		slackClient: slack.New(slackToken),
		slackToken:  slackToken,
	}
	bot.callbacks = map[string]CallbackFunc{
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
func (bot *TadoBot) Run() {
	bot.slackRTM = bot.slackClient.NewRTM()
	go bot.slackRTM.ManageConnection()

loop:
	for msg := range bot.slackRTM.IncomingEvents {
		channel, attachments, stop := bot.processEvent(msg)

		for _, attachment := range attachments {
			if _, _, err := bot.slackRTM.PostMessage(
				channel,
				slack.MsgOptionAttachments(attachment),
				slack.MsgOptionAsUser(true),
			); err != nil {
				log.WithField("err", err).Warning("failed to send on slack")
			}
		}

		if stop {
			break loop
		}
	}
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
		if attachment := bot.processMessage(ev.Text); attachment != nil {
			attachments = append(attachments, *attachment)
		}
	// case *slack.RTMError:
	//	log.WithField("error", ev.Error()).Error("Error")
	//	// TODO: reconnect here?
	case *slack.InvalidAuthEvent:
		log.Error("invalid credentials")
		stop = true
	}
	return
}

func (bot *TadoBot) processMessage(text string) (attachment *slack.Attachment) {
	// check if we're mentioned
	log.WithField("text", text).Debug("processing slack chatter")
	if parts := strings.Split(text, " "); len(parts) > 0 {
		if parts[0] == "<@"+bot.userID+">" {
			if command, ok := bot.callbacks[parts[1]]; ok {
				log.WithField("command", parts[1]).Debug("command found")
				attachment = &slack.Attachment{
					Color: "good",
					Text:  strings.Join(command(), "\n"),
				}
			} else {
				attachment = &slack.Attachment{
					Color: "warning",
					Title: "Unknown command \"" + parts[1] + "\"",
					Text:  strings.Join(bot.doHelp(), "\n"),
				}
			}
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

			// if channel.IsChannel || (channel.IsIM && channel.Conversation.ID == bot.userID) {
			channelIDs = append(channelIDs, channel.ID)
			// }
		}
	}

	log.WithFields(log.Fields{
		"channelIDs": channelIDs,
		"err":        err,
	}).Debug("found channels")

	return
}

func (bot *TadoBot) SendMessage(title, text string) (err error) {
	attachment := slack.Attachment{
		Title: title,
		Text:  text,
	}

	for _, channelID := range bot.channelIDs {
		_, _, err = bot.slackRTM.PostMessage(
			channelID,
			slack.MsgOptionAttachments(attachment),
			slack.MsgOptionAsUser(true),
		)
	}

	log.WithField("err", err).Debug("sent a message")

	return
}

func (bot *TadoBot) doHelp() []string {
	var commands = make([]string, 0)
	for command := range bot.callbacks {
		commands = append(commands, command)
	}
	return []string{"supported commands: " + strings.Join(commands, ", ")}
}

func (bot *TadoBot) doVersion() []string {
	return []string{"tado " + version.BuildVersion}
}
