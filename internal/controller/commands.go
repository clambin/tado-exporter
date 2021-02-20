package controller

import (
	"fmt"
	"github.com/slack-go/slack"
	"sort"
	"strings"
	"time"
)

func (controller *Controller) doUsers() []slack.Attachment {
	output := make([]string, 0)
	for _, device := range controller.proxy.MobileDevice {
		if device.Settings.GeoTrackingEnabled {
			state := "away"
			if device.Location.AtHome {
				state = "home"
			}
			output = append(output, fmt.Sprintf("%s: %s", device.Name, state))
		}
	}
	sort.Strings(output)
	return []slack.Attachment{
		{
			Color: "good",
			Title: "Users:",
			Text:  strings.Join(output, "\n"),
		},
	}
}

func (controller *Controller) doRooms() []slack.Attachment {
	output := make([]string, 0)
	for zoneID, zoneInfo := range controller.proxy.ZoneInfo {
		mode := ""
		if zoneInfo.Overlay.Type == "MANUAL" &&
			zoneInfo.Overlay.Setting.Type == "HEATING" {
			mode = " MANUAL"
		}
		output = append(output, fmt.Sprintf("%s: %.1fºC (target: %.1fºC%s)",
			controller.proxy.Zone[zoneID].Name,
			zoneInfo.SensorDataPoints.Temperature.Celsius,
			zoneInfo.Setting.Temperature.Celsius,
			mode,
		))
	}
	sort.Strings(output)
	return []slack.Attachment{
		{
			Color: "good",
			Title: "Rooms:",
			Text:  strings.Join(output, "\n"),
		}}
}

func (controller *Controller) doRules() (responses []slack.Attachment) {
	awayResponses := controller.doRulesAutoAway()
	limitResponses := controller.doRulesLimitOverlay()

	responses = append(responses, awayResponses[0])
	responses = append(responses, limitResponses[0])
	return
}

func (controller *Controller) doRulesAutoAway() []slack.Attachment {
	output := make([]string, 0)
	for _, entry := range controller.AutoAwayInfo {
		var response string
		switch entry.state {
		case autoAwayStateUndetermined:
			response = "undetermined"
		case autoAwayStateHome:
			response = "home"
		case autoAwayStateAway:
			response = "away. will set " + entry.Zone.Name + " to manual in " +
				entry.ActivationTime.Sub(time.Now()).Round(1*time.Minute).String()
		case autoAwayStateExpired:
			response = "away. " + entry.Zone.Name + " will be set to manual"
		case autoAwayStateReported:
			response = "away. " + entry.Zone.Name + " is set to manual"
		}
		output = append(output, entry.MobileDevice.Name+" is "+response)
	}
	sort.Strings(output)
	return []slack.Attachment{
		{
			Color: "good",
			Title: "autoAway rules:",
			Text:  strings.Join(output, "\n"),
		},
	}
}

func (controller *Controller) doRulesLimitOverlay() []slack.Attachment {
	output := make([]string, 0)
	for zoneID, entry := range controller.Overlays {
		output = append(output,
			controller.proxy.Zone[zoneID].Name+" will be reset to auto in "+entry.Sub(time.Now()).Round(1*time.Minute).String(),
		)
	}
	if len(output) > 0 {
		sort.Strings(output)
	} else {
		output = []string{"No rooms in manual control"}
	}
	return []slack.Attachment{
		{
			Color: "good",
			Title: "limitOverlay rules:",
			Text:  strings.Join(output, "\n"),
		},
	}
}
