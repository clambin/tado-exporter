package scheduler

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/model"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func (scheduler *Scheduler) makeAttachment(task *Task) (attachments []slack.Attachment) {
	zoneName := scheduler.getZoneName(task.ZoneID)

	log.WithFields(log.Fields{"zone": zoneName, "state": task.State.String()}).Info("setting zone state")

	var title string
	switch task.State.State {
	case model.Off:
		title = "switching off heating in " + zoneName
	case model.Auto:
		title = "switching off manual temperature control in " + zoneName
	case model.Manual:
		title = fmt.Sprintf("setting %s to %.1fº", zoneName, task.State.Temperature.Celsius)
	default:
		title = "unknown state detected for " + zoneName
	}
	return []slack.Attachment{{
		Color: "good",
		Title: title,
	}}
}

func (scheduler *Scheduler) getZoneName(zoneID int) (name string) {
	// TODO: cache this?
	name = "unknown"
	if zones, err := scheduler.API.GetZones(); err == nil {
		for _, zone := range zones {
			if zone.ID == zoneID {
				name = zone.Name
				break
			}
		}
	}
	return
}
