package zonemanager

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/slack-go/slack"
	"time"
)

type Task struct {
	zoneID     int
	zoneName   string
	state      models.ZoneState
	reason     string
	activation time.Time
	Cancel     chan struct{}
}

func (task *Task) Run(when time.Duration, fireChannel chan *Task) {
	timer := time.NewTicker(when)
loop:
	for {
		select {
		case <-task.Cancel:
			break loop
		case <-timer.C:
			fireChannel <- task
			break loop
		}
	}
	timer.Stop()
}

func (task *Task) notification() (attachment []slack.Attachment) {
	var text string

	switch task.state.State {
	case models.ZoneOff:
		text = "switching off heating"
	case models.ZoneAuto:
		text = "moving back to auto mode"
	case models.ZoneManual:
		text = fmt.Sprintf("setting to %.1fºC", task.state.Temperature.Celsius)
	}

	if task.activation.IsZero() == false {
		text += " in " + task.activation.Sub(time.Now()).Round(1*time.Second).String()
	}

	return []slack.Attachment{{
		Color: "good",
		Title: task.reason,
		Text:  text,
	}}
}

func (task *Task) cancelNotification(reason string) []slack.Attachment {
	return []slack.Attachment{{
		Color: "good",
		Title: "resetting rule for " + task.zoneName,
		Text:  reason,
	}}
}
