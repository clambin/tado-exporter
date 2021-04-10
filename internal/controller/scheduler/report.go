package scheduler

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/slack-go/slack"
	"strings"
	"time"
)

func (scheduler *Scheduler) reportTasks() []slack.Attachment {
	text := make([]string, 0, len(scheduler.tasks))
	for id, task := range scheduler.tasks {
		var action string
		switch task.task.State.State {
		case model.Off:
			action = "switching off heating in " + scheduler.getZoneName(id)
		case model.Auto:
			action = "setting " + scheduler.getZoneName(id) + " to auto mode"
		case model.Manual:
			action = fmt.Sprintf("setting %s to %.1fº",
				scheduler.getZoneName(id), task.task.State.Temperature.Celsius)
		}
		text = append(text, action+" in "+task.activation.Sub(time.Now()).Round(5*time.Second).String())
	}
	return []slack.Attachment{{
		Color: "good",
		Title: "rules:",
		Text:  strings.Join(text, "\n"),
	}}
}
