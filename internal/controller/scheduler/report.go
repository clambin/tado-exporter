package scheduler

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/slack-go/slack"
	"strings"
	"time"
)

func (scheduler *Scheduler) reportTasks() []slack.Attachment {
	text := make([]string, 0, len(scheduler.tasks))

	for id, scheduled := range scheduler.tasks {
		var action string
		switch scheduled.task.State.State {
		case models.ZoneOff:
			action = "switch off heating"
		case models.ZoneAuto:
			action = "set to auto mode"
		case models.ZoneManual:
			action = fmt.Sprintf("set temperature to %.1fº", scheduled.task.State.Temperature.Celsius)
		}
		text = append(text,
			scheduler.getZoneName(id)+": will "+action+" in "+
				scheduled.activation.Sub(time.Now()).Round(5*time.Second).String(),
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

	return []slack.Attachment{{
		Color: "good",
		Title: slackTitle,
		Text:  slackText,
	}}
}
