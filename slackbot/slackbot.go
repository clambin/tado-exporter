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
//     bot.GetPostChannel() <- []slack.Attachment{{Text: "Hello world"}}
package slackbot

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"sort"
	"strings"
	"sync"
)

// SlackBot interface
//go:generate mockery --name SlackBot
type SlackBot interface {
	RegisterCallback(command string, commandFunc CommandFunc)
	Run(ctx context.Context) (err error)
	Send(channel, color, title, text string) (err error)
	GetPostChannel() (ch PostChannel)
}

// Agent structure
type Agent struct {
	postChannel PostChannel
	SlackClient SlackClient
	Events      chan slack.RTMEvent

	name      string
	channels  []string
	userID    string
	callbacks map[string]CommandFunc
	connected bool
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
type CommandFunc func(ctx context.Context, args ...string) []slack.Attachment

// PostChannel to send output to slack
type PostChannel chan []slack.Attachment

// Create a slackbot
func Create(name string, slackToken string, callbacks map[string]CommandFunc) (bot *Agent) {
	eventsChannel := make(chan slack.RTMEvent, 10)

	bot = &Agent{
		postChannel: make(chan []slack.Attachment, 10),
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
func (bot *Agent) Run(ctx context.Context) (err error) {
	log.Info("slackBot started")

	if bot.channels, err = bot.SlackClient.GetChannels(); err != nil {
		return err
	}

	go bot.SlackClient.Run(ctx)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case event := <-bot.Events:
			channel, attachments := bot.processEvent(ctx, event)
			if len(attachments) > 0 {
				err = bot.sendAttachments(channel, attachments)
			}
		case attachments := <-bot.postChannel:
			err = bot.sendAttachments("", attachments)
		}

		if err != nil {
			log.WithError(err).Warning("failed to post message on Slack")
		}
	}

	// close(bot.Events)
	close(bot.postChannel)

	log.Info("slackBot stopped")
	return
}

func (bot *Agent) Send(channel, color, title, text string) (err error) {
	var channels = bot.channels
	if channel != "" {
		channels = []string{channel}
	}

	for _, c := range channels {
		message := SlackMessage{
			Channel: c,
			Attachments: []slack.Attachment{{
				Color: color,
				Title: title,
				Text:  text,
			}},
		}
		log.WithFields(log.Fields{"channel": message.Channel, "title": title, "text": text}).Debug("sending message")
		err = bot.SlackClient.Send(message)
		if err != nil {
			break
		}
	}
	return
}

func (bot *Agent) sendAttachments(channel string, attachments []slack.Attachment) (err error) {
	var channels = bot.channels
	if channel != "" {
		channels = []string{channel}
	}

	for _, c := range channels {
		log.WithFields(log.Fields{"channel": c}).Debug("sending message")
		err = bot.SlackClient.Send(SlackMessage{Channel: c, Attachments: attachments})
		if err != nil {
			break
		}
	}
	return
}

func (bot *Agent) processEvent(ctx context.Context, msg slack.RTMEvent) (channel string, attachments []slack.Attachment) {
	switch ev := msg.Data.(type) {
	// case *slack.HelloEvent:
	//	log.Debug("hello")
	case *slack.ConnectedEvent:
		bot.userID = ev.Info.User.ID
		if !bot.connected {
			log.WithField("userID", bot.userID).Info("connected to slack")
			bot.connected = true
		} else {
			log.WithField("userID", bot.userID).Debug("reconnected to slack")
		}
	case *slack.MessageEvent:
		log.WithFields(log.Fields{"name": ev.Name, "user": ev.User, "channel": ev.Channel, "type": ev.Type, "userName": ev.Username, "botID": ev.BotID}).Debug("slack message received: " + ev.Text)
		channel = ev.Channel
		attachments = bot.processMessage(ctx, ev.Text)
	case *slack.RTMError:
		log.WithField("error", ev.Error()).Error("error reading on slack RTM connection")
	case *slack.InvalidAuthEvent:
		log.Error("error received from slack: invalid credentials")
	}
	return
}

func (bot *Agent) processMessage(ctx context.Context, text string) (attachments []slack.Attachment) {
	// check if we're mentioned
	log.WithField("text", text).Debug("processing slack chatter")

	command, args := bot.parseCommand(text)
	if command == "" {
		return
	}

	callback, found := bot.getCallback(command)
	if !found {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "invalid command",
		}}
	}

	log.WithFields(log.Fields{"command": command}).Info("slackbot running command")
	attachments = callback(ctx, args...)
	log.WithFields(log.Fields{"command": command, "outputs": len(attachments)}).Debug("command run")

	return
}

func (bot *Agent) GetPostChannel() (ch PostChannel) {
	return bot.postChannel
}

func (bot *Agent) RegisterCallback(command string, callback CommandFunc) {
	bot.cbLock.Lock()
	defer bot.cbLock.Unlock()

	bot.callbacks[command] = callback
}

func (bot *Agent) getCallback(command string) (callback CommandFunc, found bool) {
	bot.cbLock.RLock()
	defer bot.cbLock.RUnlock()

	callback, found = bot.callbacks[command]
	return
}

func (bot *Agent) doHelp(_ context.Context, _ ...string) []slack.Attachment {
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

func (bot *Agent) doVersion(_ context.Context, _ ...string) []slack.Attachment {
	return []slack.Attachment{
		{
			Color: "good",
			Text:  bot.name,
		},
	}
}
