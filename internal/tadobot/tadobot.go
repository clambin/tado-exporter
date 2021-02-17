package tadobot

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"net/http"
	"strings"
)

type CallbackFunc func() []string

type TadoBot struct {
	tado.API
	slackClient *slack.Client
	slackRTM    *slack.RTM
	slackToken  string
	userID      string
	channelIDs  []string
	callbacks   map[string]CallbackFunc
}

// Create connects to a slackbot designated by token
func Create(slackToken, tadoUser, tadoPassword, tadoSecret string, callbacks map[string]CallbackFunc) (bot *TadoBot, err error) {
	bot = &TadoBot{
		API: &tado.APIClient{
			HTTPClient:   &http.Client{},
			Username:     tadoUser,
			Password:     tadoPassword,
			ClientSecret: tadoSecret,
		},
		slackClient: slack.New(slackToken),
		slackToken:  slackToken,
	}
	bot.callbacks = map[string]CallbackFunc{
		"help":    bot.doHelp,
		"version": bot.doVersion,
		"rooms":   bot.doRooms,
		"users":   bot.doUsers,
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
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			log.WithField("ev", ev).Debug("hello")
		case *slack.ConnectedEvent:
			bot.userID = ev.Info.User.ID
			log.WithField("userID", bot.userID).Info("tadoBot connected to slack")
			_ = bot.SendMessage("", "tadobot reporting for duty")
		case *slack.MessageEvent:
			log.WithFields(log.Fields{
				"name":     ev.Name,
				"user":     ev.User,
				"channel":  ev.Channel,
				"type":     ev.Type,
				"userName": ev.Username,
				"botID":    ev.BotID,
			}).Debug("message received: " + ev.Text)
			if attachment := bot.processMessage(ev.Text); attachment != nil {
				if _, _, err := bot.slackRTM.PostMessage(
					ev.Channel,
					slack.MsgOptionAttachments(*attachment),
					slack.MsgOptionAsUser(true),
				); err != nil {
					log.WithField("err", err).Warning("failed to send on slack")
				}
			}
		case *slack.RTMError:
			log.WithField("error", ev.Error()).Error("Error")
			// TODO: reconnect here?
		case *slack.InvalidAuthEvent:
			log.Error("invalid credentials")
			break loop
		}
	}
}

func (bot *TadoBot) processMessage(text string) (attachment *slack.Attachment) {
	// check if we're mentioned
	log.WithField("text", text).Debug("processing slack chatter")
	if parts := strings.Split(text, " "); len(parts) > 0 {
		if parts[0] == "<@"+bot.userID+">" {
			if command, ok := bot.callbacks[parts[1]]; ok {
				log.WithField("command", command).Debug("command found")
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
// This is either the bot's direct channel or any channels the bot's been invited to
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

func (bot *TadoBot) doRooms() (responses []string) {
	var (
		err   error
		zones []*tado.Zone
	)
	zoneInfos := make(map[string]*tado.ZoneInfo)
	if zones, err = bot.GetZones(); err == nil {
		for _, zone := range zones {
			var zoneInfo *tado.ZoneInfo
			if zoneInfo, err = bot.GetZoneInfo(zone.ID); err == nil {
				zoneInfos[zone.Name] = zoneInfo
			} else {
				break
			}
		}
	}

	if err == nil {
		for zoneName, zoneInfo := range zoneInfos {
			mode := ""
			if zoneInfo.Overlay.Type == "MANUAL" &&
				zoneInfo.Overlay.Setting.Type == "HEATING" {
				mode = " MANUAL"
			}
			responses = append(responses, fmt.Sprintf("%s: %.1fºC (target: %.1fºC%s)",
				zoneName,
				zoneInfo.SensorDataPoints.Temperature.Celsius,
				zoneInfo.Setting.Temperature.Celsius,
				mode,
			))
		}
	} else {
		responses = []string{"unable to get rooms: " + err.Error()}
	}

	return
}

func (bot *TadoBot) doUsers() (responses []string) {
	var (
		err     error
		devices []*tado.MobileDevice
	)
	if devices, err = bot.GetMobileDevices(); err == nil {
		for _, device := range devices {
			if device.Settings.GeoTrackingEnabled {
				state := "away"
				if device.Location.AtHome {
					state = "home"
				}
				responses = append(responses, fmt.Sprintf("%s: %s", device.Name, state))
			}
		}
	} else {
		responses = []string{"unable to get users: " + err.Error()}
	}
	return
}
