package controller

import (
	"github.com/clambin/tado"
	"github.com/slack-go/slack"
	"strings"
	"time"
)

func (controller *Controller) ReportTasks(_ ...string) (attachments []slack.Attachment) {
	controller.Report <- struct{}{}
	return
}

func (controller *Controller) reportTasks(_ ...string) {
	text := make([]string, 0)
	for _, task := range controller.scheduler.GetAllScheduled() {
		state := task.Args[1].(tado.ZoneState)
		var action string
		switch state {
		case tado.ZoneStateOff:
			action = "switching off heating"
		case tado.ZoneStateAuto:
			action = "moving to auto mode"
		case tado.ZoneStateManual:
			action = "setting to manual temperature control"
		}

		name, ok := controller.cache.GetZoneName(int(task.ID))
		if ok == false {
			name = "unknown"
		}
		text = append(text, name+": "+action+" in "+
			task.Activation.Sub(time.Now()).Round(1*time.Second).String(),
		)
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
