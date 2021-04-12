package zonemanager

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/models"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"time"
)

func (mgr *Manager) notifyPendingTask(task *Task) (attachments []slack.Attachment) {
	zoneName := mgr.getZoneName(task.zoneID)

	log.WithFields(log.Fields{"zone": zoneName, "state": task.state.String()}).Debug("queuing zone state change")

	var title, text string
	title = "unknown state detected for " + zoneName
	delta := task.activation.Sub(time.Now()).Round(1 * time.Second).String()

	switch task.state.State {
	case models.ZoneOff:
		title = zoneName + " users not home"
		text = "switching off heating in " + delta
	case models.ZoneAuto:
		title = "manual temperature setting detected in " + zoneName
		text = "will move back to auto mode in " + delta
	case models.ZoneManual:
		title = "setting " + zoneName + " to manual temperature setting"
		text = fmt.Sprintf("setting to %.1fº in %s",
			task.state.Temperature.Celsius,
			delta,
		)
	}

	return []slack.Attachment{{
		Color: "good",
		Title: title,
		Text:  text,
	}}
}

func (mgr *Manager) notifyExecutedTask(task *Task) (attachments []slack.Attachment) {
	zoneName := mgr.getZoneName(task.zoneID)

	log.WithFields(log.Fields{"zone": zoneName, "state": task.state.String()}).Info("setting zone state")

	var title, text string
	title = "unknown state detected for " + zoneName
	switch task.state.State {
	case models.ZoneOff:
		title = "switching off heating in " + zoneName
		text = "users not home"
	case models.ZoneAuto:
		title = "Setting " + zoneName + " back to auto mode"
		text = "overlay expired"
	case models.ZoneManual:
		title = "setting " + zoneName + " to manual temperature"
		text = fmt.Sprintf("setting to %.1fº", task.state.Temperature.Celsius)
	}
	return []slack.Attachment{{
		Color: "good",
		Title: title,
		Text:  text,
	}}
}

func (mgr *Manager) getZoneName(zoneID int) (name string) {
	var ok bool
	if name, ok = mgr.nameCache[zoneID]; ok == false {
		name = "unknown"
		if zones, err := mgr.API.GetZones(); err == nil {
			for _, zone := range zones {
				if zone.ID == zoneID {
					name = zone.Name
					mgr.nameCache[zoneID] = name
					break
				}
			}
		}
	}
	return
}
