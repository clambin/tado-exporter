package scheduler

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/models"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"time"
)

func (scheduler *Scheduler) notifyPendingTask(task *Task) (attachments []slack.Attachment) {
	zoneName := scheduler.getZoneName(task.ZoneID)

	log.WithFields(log.Fields{"zone": zoneName, "state": task.State.String()}).Debug("queuing zone state change")

	var title, text string
	title = "unknown state detected for " + zoneName
	switch task.State.State {
	case models.ZoneOff:
		title = zoneName + " users not home"
		text = "switching off heating in " + task.When.String()
	case models.ZoneAuto:
		title = "manual temperature setting detected in " + zoneName
		text = "will move back to auto mode in " + task.When.Round(1*time.Minute).String()
	case models.ZoneManual:
		title = "setting " + zoneName + " to manual temperature setting"
		text = fmt.Sprintf("setting to %.1fº in %s",
			task.State.Temperature.Celsius,
			task.When.String(),
		)
	}

	return []slack.Attachment{{
		Color: "good",
		Title: title,
		Text:  text,
	}}
}

func (scheduler *Scheduler) notifyExecutedTask(task *Task) (attachments []slack.Attachment) {
	zoneName := scheduler.getZoneName(task.ZoneID)

	log.WithFields(log.Fields{"zone": zoneName, "state": task.State.String()}).Info("setting zone state")

	var title, text string
	title = "unknown state detected for " + zoneName
	switch task.State.State {
	case models.ZoneOff:
		title = "switching off heating in " + zoneName
		text = "users not home"
	case models.ZoneAuto:
		title = "Setting " + zoneName + " back to auto mode"
		text = "overlay expired"
	case models.ZoneManual:
		title = "setting " + zoneName + " to manual temperature"
		text = fmt.Sprintf("setting to %.1fº", task.State.Temperature.Celsius)
	}
	return []slack.Attachment{{
		Color: "good",
		Title: title,
		Text:  text,
	}}
}

func (scheduler *Scheduler) getZoneName(zoneID int) (name string) {
	var ok bool
	if name, ok = scheduler.nameCache[zoneID]; ok == false {
		name = "unknown"
		if zones, err := scheduler.API.GetZones(); err == nil {
			for _, zone := range zones {
				if zone.ID == zoneID {
					name = zone.Name
					scheduler.nameCache[zoneID] = name
					break
				}
			}
		}
	}
	return
}
