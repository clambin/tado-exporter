package controller

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/slack-go/slack"
	"strings"
	"time"
)

func (controller *Controller) ReportRules(_ ...string) (attachments []slack.Attachment) {
	go controller.reportRules()
	return
}

func (controller *Controller) ReportRooms(_ ...string) (attachments []slack.Attachment) {
	go controller.reportRooms()
	return
}

func (controller *Controller) reportRules(_ ...string) {
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

	controller.PostChannel <- []slack.Attachment{{
		Color: "good",
		Title: slackTitle,
		Text:  slackText,
	}}
}

func (controller *Controller) reportRooms(_ ...string) {
	text := make([]string, 0)

	for _, zoneID := range controller.cache.GetZones() {
		name, ok := controller.cache.GetZoneName(zoneID)

		var temperature, targetTemperature float64
		var zoneState tado.ZoneState
		if ok {
			temperature, targetTemperature, zoneState, ok = controller.cache.GetZoneInfo(zoneID)
		}

		if ok {
			var stateStr string
			switch zoneState {
			case tado.ZoneStateOff:
				stateStr = "off"
			case tado.ZoneStateAuto:
				stateStr = fmt.Sprintf("target: %.1f", targetTemperature)
			case tado.ZoneStateTemporaryManual:
				stateStr = fmt.Sprintf("target: %.1f, MANUAL", targetTemperature)
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

	controller.PostChannel <- []slack.Attachment{{
		Color: slackColor,
		Title: slackTitle,
		Text:  slackText,
	}}
}
