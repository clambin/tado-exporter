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
	slackToken  string
	userID      string
	channels    []string
	callbacks   map[string]CallbackFunc
}

// Create connects to a slackbot designated by token
func Create(slackToken, tadoUser, tadoPassword, tadoSecret string) (bot *TadoBot, err error) {
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
	return
}

// Run the slackbot
func (bot *TadoBot) Run() {
	rtm := bot.slackClient.NewRTM()
	go rtm.ManageConnection()

loop:
	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			log.WithField("ev", ev).Debug("hello")
		case *slack.ConnectedEvent:
			bot.userID = ev.Info.User.ID
			log.WithField("userID", bot.userID).Info("tadoBot connected to slack")
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
				if _, _, err := rtm.PostMessage(
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

func (bot *TadoBot) doHelp() []string {
	var commands = make([]string, 0)
	for command := range bot.callbacks {
		commands = append(commands, command)
	}
	return []string{"supported commands: " + strings.Join(commands, ", ")}
}

func (bot *TadoBot) doVersion() []string {
	return []string{"tado v" + version.BuildVersion}
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
