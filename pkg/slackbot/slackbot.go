package slackbot

import (
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"sort"
	"strings"
)

type CommandFunc func(args ...string) []slack.Attachment
type PostChannel chan []slack.Attachment

type SlackBot struct {
	PostChannel PostChannel

	name        string
	slackClient *SlackClient
	events      chan slack.RTMEvent
	messages    chan Message
	userID      string
	callbacks   map[string]CommandFunc
	reconnect   bool
}

// Create connects to a slackbot designated by token
func Create(name string, slackToken string, callbacks map[string]CommandFunc) (bot *SlackBot, err error) {
	bot = &SlackBot{
		PostChannel: make(chan []slack.Attachment, 5),
		name:        name,
		events:      make(chan slack.RTMEvent),
		messages:    make(chan Message),
	}
	if slackToken != "" {
		bot.slackClient = NewClient(slackToken, bot.events, bot.messages)
	}
	bot.callbacks = map[string]CommandFunc{
		"help":    bot.doHelp,
		"version": bot.doVersion,
	}
	for cmd, callbackFunction := range callbacks {
		bot.callbacks[cmd] = callbackFunction
	}
	return
}

// Run the slackbot
func (bot *SlackBot) Run() (err error) {
	if bot.slackClient != nil {
		go bot.slackClient.Run()
	}

loop:
	for {
		var (
			channel     string
			attachments []slack.Attachment
			stop        bool
		)

		select {
		case event := <-bot.events:
			channel, attachments, stop = bot.processEvent(event)
			if stop {
				break loop
			}
		case attachments = <-bot.PostChannel:
		}

		if len(attachments) > 0 {
			bot.messages <- Message{Channel: channel, Attachments: attachments}
		}
	}

	close(bot.events)
	close(bot.messages)
	close(bot.PostChannel)

	return
}

func (bot *SlackBot) processEvent(msg slack.RTMEvent) (channel string, attachments []slack.Attachment, stop bool) {
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

func (bot *SlackBot) processMessage(text string) (attachments []slack.Attachment) {
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

func (bot *SlackBot) doHelp(_ ...string) []slack.Attachment {
	var commands = make([]string, 0)
	for command := range bot.callbacks {
		commands = append(commands, command)
	}
	sort.Strings(commands)

	return []slack.Attachment{
		{
			Color: "good",
			Title: "supported commands",
			Text:  strings.Join(commands, ", "),
		},
	}
}

func (bot *SlackBot) doVersion(_ ...string) []slack.Attachment {
	return []slack.Attachment{
		{
			Color: "good",
			Text:  bot.name,
		},
	}
}
