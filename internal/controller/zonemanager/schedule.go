package zonemanager

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"time"
)

func (mgr *Manager) scheduleZoneStateChange(ctx context.Context, zoneID int, state models.ZoneState, when time.Duration, reason string) {
	// when does this task run?
	activation := time.Now().Add(when)

	// check if we already have a task running for the zoneID
	running, ok := mgr.scheduler.GetScheduled(scheduler.TaskID(zoneID))

	if ok {
		// if that change is already pending for that room, and it will start earlier, don't schedule the new change
		if state.State == running.Args[1].(models.ZoneState).State && activation.After(running.Activation) { // running.Sub(activation).Round(time.Minute) <= 0 {
			log.WithFields(log.Fields{"zone": zoneID, "running": running, "new": activation}).Debug("earlier task already found. won't schedule this task")
			return
		}

		// cancel the running task
		mgr.scheduler.Cancel(scheduler.TaskID(zoneID))
		log.WithField("zone", zoneID).Debug("canceled running task")
	}

	mgr.scheduler.Schedule(ctx, scheduler.Task{
		ID:  scheduler.TaskID(zoneID),
		Run: mgr.runTask,
		Args: []interface{}{
			zoneID,
			state,
			reason,
		},
		When: when,
	})

	log.WithFields(log.Fields{
		"zone":  zoneID,
		"state": state.String(),
		"when":  when.String(),
	}).Debug("scheduled task")

	// post slack notification, if it will be executed later
	if when > 0 && mgr.postChannel != nil {
		ann := mgr.getNotification(state, reason, activation)
		// log.Infof("sending notification: %v", ann)
		mgr.postChannel <- ann
	}
}

func (mgr *Manager) cancelZoneStateChange(zoneID int, reason string) {
	if _, found := mgr.scheduler.GetScheduled(scheduler.TaskID(zoneID)); found {
		mgr.scheduler.Cancel(scheduler.TaskID(zoneID))

		if mgr.postChannel != nil {
			ann := mgr.getCancelNotification(zoneID, reason)
			// log.Infof("sending notification: %v", ann)
			mgr.postChannel <- ann
		}
	}
}

func (mgr *Manager) runTask(ctx context.Context, args []interface{}) {
	zoneID := args[0].(int)
	state := args[1].(models.ZoneState)
	reason := args[2].(string)

	var err error
	switch state.State {
	case models.ZoneOff:
		err = mgr.API.SetZoneOverlay(ctx, zoneID, 5.0)
	case models.ZoneAuto:
		err = mgr.API.DeleteZoneOverlay(ctx, zoneID)
	case models.ZoneManual:
		err = mgr.API.SetZoneOverlay(ctx, zoneID, state.Temperature.Celsius)
	}

	if err != nil {
		log.WithField("err", err).Warning("unable to update zone")
		return
	}

	if mgr.postChannel != nil {
		mgr.postChannel <- mgr.getNotification(state, reason, time.Time{})
	}

	log.WithFields(log.Fields{"zone": zoneID, "state": state.String()}).Debug("executed task")
}

func (mgr *Manager) getNotification(state models.ZoneState, reason string, activation time.Time) (attachment []slack.Attachment) {
	var text string

	switch state.State {
	case models.ZoneOff:
		text = "switching off heating"
	case models.ZoneAuto:
		text = "moving to auto mode"
	case models.ZoneManual:
		text = fmt.Sprintf("setting to %.1fºC", state.Temperature.Celsius)
	}

	if activation.IsZero() == false {
		text += " in " + activation.Sub(time.Now()).Round(1*time.Second).String()
	}

	return []slack.Attachment{{
		Color: "good",
		Title: reason,
		Text:  text,
	}}
}

func (mgr *Manager) getCancelNotification(zoneID int, reason string) []slack.Attachment {
	_, zoneName, _ := mgr.stateManager.LookupZone(zoneID, "")
	return []slack.Attachment{{
		Color: "good",
		Title: "resetting rule for " + zoneName,
		Text:  reason,
	}}
}
