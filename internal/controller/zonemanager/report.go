package zonemanager

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/slack-go/slack"
	"strings"
	"time"
)

func (mgr *Manager) ReportTasks(_ ...string) (attachments []slack.Attachment) {
	mgr.Report <- struct{}{}
	return
}

func (mgr *Manager) reportTasks(_ ...string) {
	text := make([]string, 0, len(mgr.tasks))

	for id, task := range mgr.tasks {
		var action string
		switch task.state.State {
		case models.ZoneOff:
			action = "switching off heating"
		case models.ZoneAuto:
			action = "moving to auto mode"
		case models.ZoneManual:
			action = fmt.Sprintf("set temperature to %.1fº", task.state.Temperature.Celsius)
		}
		text = append(text,
			mgr.getZoneName(id)+": "+action+" in "+
				task.activation.Sub(time.Now()).Round(1*time.Second).String(),
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

	mgr.postChannel <- []slack.Attachment{{
		Color: "good",
		Title: slackTitle,
		Text:  slackText,
	}}
}
