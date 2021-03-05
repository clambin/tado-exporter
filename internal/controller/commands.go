package controller

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/commands"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/slack-go/slack"
	"sort"
	"strconv"
	"strings"
)

func (controller *Controller) doUsers(_ ...string) []slack.Attachment {
	var (
		err           error
		mobileDevices []*tado.MobileDevice
	)
	output := make([]string, 0)
	if mobileDevices, err = controller.GetMobileDevices(); err == nil {
		for _, device := range mobileDevices {
			if device.Settings.GeoTrackingEnabled {
				state := "away"
				if device.Location.AtHome {
					state = "home"
				}
				if device.Location.Stale {
					state += " (stale)"
				}
				output = append(output, fmt.Sprintf("%s: %s", device.Name, state))
			}
		}
		sort.Strings(output)
	}
	return []slack.Attachment{
		{
			Color: "good",
			Title: "Users:",
			Text:  strings.Join(output, "\n"),
		},
	}
}

func (controller *Controller) doRooms(_ ...string) []slack.Attachment {
	var (
		err      error
		zones    []*tado.Zone
		zoneInfo *tado.ZoneInfo
	)
	output := make([]string, 0)

	if zones, err = controller.GetZones(); err == nil {
		for _, zone := range zones {
			if zoneInfo, err = controller.GetZoneInfo(zone.ID); err == nil {

				mode := ""
				if zoneInfo.Overlay.Type == "MANUAL" &&
					zoneInfo.Overlay.Setting.Type == "HEATING" {
					mode = " MANUAL"
				}

				output = append(output, fmt.Sprintf("%s: %.1fºC (target: %.1fºC%s)",
					zone.Name,
					zoneInfo.SensorDataPoints.Temperature.Celsius,
					zoneInfo.Setting.Temperature.Celsius,
					mode,
				))
			}
		}
		sort.Strings(output)
	}
	return []slack.Attachment{
		{
			Color: "good",
			Title: "Rooms:",
			Text:  strings.Join(output, "\n"),
		},
	}
}

func (controller *Controller) doRules(args ...string) (responses []slack.Attachment) {
	awayResponses := controller.doRulesAutoAway(args...)
	limitResponses := controller.doRulesLimitOverlay(args...)

	responses = append(responses, awayResponses[0])
	responses = append(responses, limitResponses[0])
	return
}

func (controller *Controller) doRulesAutoAway(_ ...string) []slack.Attachment {
	command := commands.Command{
		Command:  1,
		Response: make(commands.ResponseChannel),
	}

	controller.autoAway.Commands <- command
	output := <-command.Response

	return []slack.Attachment{
		{
			Color: "good",
			Title: "autoAway rules:",
			Text:  strings.Join(output, "\n"),
		},
	}
}

func (controller *Controller) doRulesLimitOverlay(_ ...string) []slack.Attachment {
	command := commands.Command{
		Command:  1,
		Response: make(commands.ResponseChannel),
	}

	controller.limiter.Commands <- command
	output := <-command.Response

	return []slack.Attachment{
		{
			Color: "good",
			Title: "limitOverlay rules:",
			Text:  strings.Join(output, "\n"),
		},
	}
}

func (controller *Controller) parseSetTemperatureArguments(args ...string) (ok bool, output slack.Attachment, zoneID int, auto bool, temperature float64) {
	if len(args) != 2 {
		output = slack.Attachment{
			Color: "bad",
			Text:  "invalid command:  set <room name> auto|<temperature>",
		}
		return
	}

	var (
		zones []*tado.Zone
		err   error
	)

	zoneName := strings.ToLower(args[0])
	if zones, err = controller.GetZones(); err == nil {
		for _, zone := range zones {
			if strings.ToLower(zone.Name) == zoneName {
				zoneID = zone.ID
				break
			}
		}
	}
	if zoneID == 0 {
		output = slack.Attachment{
			Color: "bad",
			Text:  "unknown room name: " + zoneName,
		}
		return
	}

	if strings.ToLower(args[1]) == "auto" {
		auto = true
	} else {
		if temperature, err = strconv.ParseFloat(args[1], 64); err != nil {
			output = slack.Attachment{
				Color: "bad",
				Text:  "invalid temperature: " + args[1],
			}
			return
		}
	}

	ok = true
	return
}

func (controller *Controller) doSetTemperature(args ...string) (output []slack.Attachment) {
	ok, errorOutput, zoneID, auto, temperature := controller.parseSetTemperatureArguments(args...)

	if ok {
		var roomCommand tadosetter.RoomCommand
		if auto {
			roomCommand = tadosetter.RoomCommand{ZoneID: zoneID, Auto: true}
			output = append(output, slack.Attachment{
				Color: "good",
				Text:  "setting " + args[0] + " back to auto",
			})
		} else {
			roomCommand = tadosetter.RoomCommand{ZoneID: zoneID, Auto: false, Temperature: temperature}
			output = append(output, slack.Attachment{
				Color: "good",
				Text:  "setting temperature in " + args[0] + " to " + args[1],
			})
		}
		controller.roomSetter.ZoneSetter <- roomCommand
	} else {
		output = append(output, errorOutput)
	}
	return
}
