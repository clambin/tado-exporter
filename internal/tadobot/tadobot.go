package tadobot

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

type CallbackFunc func() string

type TadoBot struct {
	tado.API
	slackbot  *slackbot.SlackBot
	channel   string
	callbacks map[string]CallbackFunc
}

// Create connects to a slackbot designated by token
func Create(slackToken, tadoUser, tadoPassword, tadoSecret string) (bot *TadoBot, err error) {
	var botHandle *slackbot.SlackBot

	if botHandle, err = slackbot.Connect(slackToken); err == nil {
		bot = &TadoBot{
			API: &tado.APIClient{
				HTTPClient:   &http.Client{},
				Username:     tadoUser,
				Password:     tadoPassword,
				ClientSecret: tadoSecret,
			},
			slackbot: botHandle,
		}
		bot.callbacks = map[string]CallbackFunc{
			"help":    bot.DoHelp,
			"version": bot.DoVersion,
			"rooms":   bot.DoRooms,
			"users":   bot.DoUsers,
		}
	}
	return
}

func (bot *TadoBot) Run() {
	var (
		err      error
		message  slackbot.Message
		response string
	)

	for {
		message, err = bot.slackbot.GetMessage()

		if err != nil {
			log.WithField("err", err).Warning("failed to get slack message")
		} else {
			if message.Type == "hello" {
				bot.channel = message.Channel
			} else if message.Type == "message" {
				if f, ok := bot.callbacks[message.Text]; ok {
					response = f()
				} else {
					response = "unknown command"
				}

				message.Text = response

				if err = bot.slackbot.PostMessage(message); err != nil {
					log.WithField("err", err).Warning("failed to send slack message")
				}
			} else {
				log.WithField("type", message.Type).Info("unhandled message type")
			}
		}
	}
}

func (bot *TadoBot) PostMessage(message string) error {
	m := slackbot.Message{
		Type:    "message",
		Channel: bot.channel,
		Text:    message,
	}
	return bot.slackbot.PostMessage(m)
}

func (bot *TadoBot) DoHelp() string {
	var commands = make([]string, 0)
	for command := range bot.callbacks {
		commands = append(commands, command)
	}
	return "supported commands: " + strings.Join(commands, ", ")
}

func (bot *TadoBot) DoVersion() string {
	return "tado v" + version.BuildVersion
}

func (bot *TadoBot) DoRooms() (response string) {
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
			if response != "" {
				response += "\n"
			}
			mode := ""
			if zoneInfo.Overlay.Type == "MANUAL" &&
				zoneInfo.Overlay.Setting.Type == "HEATING" {
				mode = " MANUAL"
			}
			response += fmt.Sprintf("%s: %.1fºC (target: %.1fºC%s)",
				zoneName,
				zoneInfo.SensorDataPoints.Temperature.Celsius,
				zoneInfo.Setting.Temperature.Celsius,
				mode,
			)
		}
	} else {
		response = "unable to get rooms: " + err.Error()
	}

	return
}

func (bot *TadoBot) DoUsers() (response string) {
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
				if response != "" {
					response += "\n"
				}
				response += fmt.Sprintf("%s: %s", device.Name, state)
			}
		}
	} else {
		response = "unable to get users: " + err.Error()
	}
	return response
}
