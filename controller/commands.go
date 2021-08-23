package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/slack-go/slack"
	"strconv"
	"strings"
	"time"
)

func (controller *Controller) ReportRules(_ context.Context, _ ...string) (attachments []slack.Attachment) {
	text := make([]string, 0)
	for _, task := range controller.scheduler.GetAllScheduled() {
		state := task.Args[1].(tado.ZoneState)
		var action string
		switch state {
		case tado.ZoneStateOff:
			action = "switching off heating"
		case tado.ZoneStateAuto:
			action = "moving to auto mode"
			//case tado.ZoneStateManual:
			//	action = "setting to manual temperature control"
		}

		name, _ := controller.cache.GetZoneName(int(task.ID))

		text = append(text,
			name+": "+action+" in "+task.Activation.Sub(time.Now()).Round(1*time.Second).String())
	}

	var slackText, slackTitle string
	if len(text) > 0 {
		slackTitle = "rules:"
		slackText = strings.Join(text, "\n")
	} else {
		slackTitle = ""
		slackText = "no rules have been triggered"
	}

	return []slack.Attachment{{
		Color: "good",
		Title: slackTitle,
		Text:  slackText,
	}}
}

func (controller *Controller) ReportRooms(_ context.Context, _ ...string) (attachments []slack.Attachment) {
	text := make([]string, 0)

	for _, zoneID := range controller.cache.GetZones() {
		name, ok := controller.cache.GetZoneName(zoneID)

		var temperature, targetTemperature float64
		var zoneState tado.ZoneState
		var duration time.Duration
		if ok {
			temperature, targetTemperature, zoneState, duration, ok = controller.cache.GetZoneInfo(zoneID)
		}

		if ok {
			var stateStr string
			switch zoneState {
			case tado.ZoneStateOff:
				stateStr = "off"
			case tado.ZoneStateAuto:
				stateStr = fmt.Sprintf("target: %.1f", targetTemperature)
			case tado.ZoneStateTemporaryManual:
				stateStr = fmt.Sprintf("target: %.1f, MANUAL for %s", targetTemperature, duration.String())
			case tado.ZoneStateManual:
				stateStr = fmt.Sprintf("target: %.1f, MANUAL", targetTemperature)
			}

			text = append(text, fmt.Sprintf("%s: %.1fºC (%s)", name, temperature, stateStr))
		}
	}

	slackColor := "bad"
	slackTitle := ""
	slackText := "no rooms found"

	if len(text) > 0 {
		slackColor = "good"
		slackTitle = "rooms:"
		slackText = strings.Join(text, "\n")
	}

	return []slack.Attachment{{
		Color: slackColor,
		Title: slackTitle,
		Text:  slackText,
	}}
}

func (controller *Controller) SetRoom(ctx context.Context, args ...string) (attachments []slack.Attachment) {
	zoneID, zoneName, temperature, duration, err := controller.parseSetCommand(args...)

	if err != nil {
		err = fmt.Errorf("invalid command: %v", err)
	}

	if err == nil {
		err = controller.API.SetZoneOverlayWithDuration(ctx, zoneID, temperature, duration)

		if err != nil {
			err = fmt.Errorf("unable to set temperature for %s: %v", zoneName, err)
		}

		controller.refresh()
	}

	if err != nil {
		attachments = []slack.Attachment{{
			Color: "bad",
			Title: "",
			Text:  err.Error(),
		}}
	} else {
		text := fmt.Sprintf("Setting target temperature for %s to %.1fºC", zoneName, temperature)
		if duration > 0 {
			text += " for " + duration.String()
		}
		attachments = []slack.Attachment{{
			Color: "good",
			Title: "",
			Text:  text,
		}}
	}

	return
}

func (controller *Controller) parseSetCommand(args ...string) (zoneID int, zoneName string, temperature float64, duration time.Duration, err error) {
	if len(args) < 1 {
		err = fmt.Errorf("missing room name")
		return
	}

	zoneName = args[0]

	var ok bool
	zoneID, _, ok = controller.cache.LookupZone(0, zoneName)

	if ok == false {
		err = fmt.Errorf("invalid room name")
		return
	}

	if len(args) < 2 {
		err = fmt.Errorf("missing target temperature")
		return
	}

	temperature, err = strconv.ParseFloat(args[1], 64)

	if err != nil {
		err = fmt.Errorf("invalid target temperature: \"%s\"", args[1])
		return
	}

	if len(args) > 2 {
		duration, err = time.ParseDuration(args[2])

		if err != nil {
			err = fmt.Errorf("invalid duration: \"%s\"", args[2])
		}
	}

	return
}

func (controller *Controller) DoRefresh(_ context.Context, _ ...string) (attachments []slack.Attachment) {
	controller.refresh()
	return
}
