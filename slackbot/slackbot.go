// Package slackbot provides a basic slackbot implementation.
// Using this package typically involves creating an Bot as follows:
//
//     bot := slackbot.New(botName, slackToken, callbacks)
//     go bot.Run()
//
// Once running, the bot will listen for any commands specified on the channel and execute them. Slackbot itself
// implements two commands: "version" (which responds with botName) and "help" (which shows all implemented commands).
// Additional commands can be added through the callbacks parameter (see Create & CommandFunc):
//
//     func doHello(args ...string) []slack.Attachment {
//	       return []slack.Attachment{{Text: "hello world " + strings.Join(args, ", ")}}
//     }
//
// The returned attachments will be sent to the slack channel where the command was issued.
//
// Additionally, output can be sent to the slack channel(s) using PostChannel, e.g.:
//
//     bot.PostChannel <- []slack.Attachment{{Text: "Hello world"}}
package slackbot

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"sort"
	"strings"
	"sync"
)

// SlackBot structure
type SlackBot struct {
	PostChannel PostChannel
	SlackClient ClientAPI
	Events      chan slack.RTMEvent

	name      string
	channels  []string
	userID    string
	callbacks map[string]CommandFunc
	reconnect bool
	cbLock    sync.RWMutex
}

// CommandFunc signature for command callback functions
//
// args will contain any additional tokens after the command, e.g.:
//
//      @slackbot say 1 2 3
//
// args will be []string{"1", "2", "3"}
//
// returns a slice of Attachments, which will be sent to slack as one message
type CommandFunc func(args ...string) []slack.Attachment

// PostChannel to send output to slack
type PostChannel chan []slack.Attachment

// Create a slackbot
func Create(name string, slackToken string, callbacks map[string]CommandFunc) (bot *SlackBot, err error) {
	eventsChannel := make(chan slack.RTMEvent, 10)

	bot = &SlackBot{
		PostChannel: make(chan []slack.Attachment, 10),
		name:        name,
		Events:      eventsChannel,
		SlackClient: newClient(slackToken, eventsChannel),
	}

	bot.callbacks = map[string]CommandFunc{
		"help":    bot.doHelp,
		"version": bot.doVersion,
	}
	for cmd, callbackFunction := range callbacks {
		bot.RegisterCallback(cmd, callbackFunction)
	}

	return
}

// Run the slackbot
func (bot *SlackBot) Run(ctx context.Context) (err error) {
	log.Info("slackBot started")

	if bot.channels, err = bot.SlackClient.GetChannels(); err != nil {
		return err
	}

	go bot.SlackClient.Run(ctx)

loop:
	for {
		var (
			channel     string
			attachments []slack.Attachment
			stop        bool
		)

		select {
		case <-ctx.Done():
			break loop
		case event := <-bot.Events:
			channel, attachments, stop = bot.processEvent(event)
			if stop {
				break loop
			}
			if len(attachments) > 0 {
				err = bot.Send(SlackMessage{Channel: channel, Attachments: attachments})
			}
		case attachments = <-bot.PostChannel:
			err = bot.Send(SlackMessage{Attachments: attachments})
		}

		if err != nil {
			log.WithError(err).Warning("failed to post message on Slack")
		}
	}

	close(bot.Events)
	close(bot.PostChannel)

	log.Info("slackBot stopped")
	return
}

func (bot *SlackBot) Send(message SlackMessage) (err error) {
	var channels []string
	if message.Channel != "" {
		channels = []string{message.Channel}
	} else {
		channels = bot.channels
	}

	for _, channel := range channels {
		message.Channel = channel
		log.WithFields(log.Fields{"channel": message.Channel}).Debug("sending message")
		if err = bot.SlackClient.Send(message); err != nil {
			break
		}
	}
	return
}

func (bot *SlackBot) processEvent(msg slack.RTMEvent) (channel string, attachments []slack.Attachment, stop bool) {
	switch ev := msg.Data.(type) {
	case *slack.HelloEvent:
		log.Debug("hello")
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
	if command == "" {
		return
	}

	callback, ok := bot.getCallback(command)
	if ok == false {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "invalid command",
		}}
	}

	log.WithFields(log.Fields{"command": command}).Info("slackbot running command")
	attachments = callback(args...)
	log.WithFields(log.Fields{"command": command, "outputs": len(attachments)}).Debug("command run")

	return
}

func (bot *SlackBot) RegisterCallback(command string, callback CommandFunc) {
	bot.cbLock.Lock()
	defer bot.cbLock.Unlock()

	bot.callbacks[command] = callback
}

func (bot *SlackBot) getCallback(command string) (callback CommandFunc, ok bool) {
	bot.cbLock.RLock()
	defer bot.cbLock.RUnlock()

	callback, ok = bot.callbacks[command]
	return
}

func (bot *SlackBot) doHelp(_ ...string) []slack.Attachment {
	bot.cbLock.RLock()
	defer bot.cbLock.RUnlock()

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
