package zonemanager

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/scheduler"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"time"
)

func (mgr *Manager) scheduleZoneStateChange(ctx context.Context, zoneID int, state tado.ZoneState, when time.Duration, reason string) {
	// when does this task run?
	activation := time.Now().Add(when)

	// check if we already have a task running for the zoneID
	running, ok := mgr.scheduler.GetScheduled(scheduler.TaskID(zoneID))

	if ok {
		// if that change is already pending for that room, and the scheduled change will start earlier, don't schedule the new change
		if state == running.Args[1].(tado.ZoneState) && activation.After(running.Activation) { // running.Sub(activation).Round(time.Minute) <= 0 {
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
		"state": state,
		"when":  when.String(),
	}).Debug("scheduled task")

	// post slack notification, if it will be executed later
	if when > 0 && mgr.PostChannel != nil {
		ann := mgr.getNotification(state, reason, activation)
		// log.Infof("sending notification: %v", ann)
		mgr.PostChannel <- ann
	}
}

func (mgr *Manager) cancelZoneStateChange(zoneID int, reason string) {
	if _, found := mgr.scheduler.GetScheduled(scheduler.TaskID(zoneID)); found {
		mgr.scheduler.Cancel(scheduler.TaskID(zoneID))

		if mgr.PostChannel != nil {
			ann := mgr.getCancelNotification(zoneID, reason)
			// log.Infof("sending notification: %v", ann)
			mgr.PostChannel <- ann
		}
	}
}

func (mgr *Manager) runTask(ctx context.Context, args []interface{}) {
	zoneID := args[0].(int)
	state := args[1].(tado.ZoneState)
	reason := args[2].(string)

	var err error
	switch state {
	case tado.ZoneStateOff:
		err = mgr.API.SetZoneOverlay(ctx, zoneID, 5.0)
	case tado.ZoneStateAuto:
		err = mgr.API.DeleteZoneOverlay(ctx, zoneID)
	//case tado.ZoneStateManual:
	//	not implemented: GetNextState never returns ZoneStateManual
	//	err = mgr.API.SetZoneOverlay(ctx, zoneID, 15.0 /* TODO: state.Temperature.Celsius */)
	default:
		panic("not implemented")
	}

	if err != nil {
		log.WithField("err", err).Warning("unable to update zone")
		return
	}

	if mgr.PostChannel != nil {
		mgr.PostChannel <- mgr.getNotification(state, reason, time.Time{})
	}

	log.WithFields(log.Fields{"zone": zoneID, "state": state}).Debug("executed task")
}

func (mgr *Manager) getNotification(state tado.ZoneState, reason string, activation time.Time) (attachment []slack.Attachment) {
	var text string

	switch state {
	case tado.ZoneStateOff:
		text = "switching off heating"
	case tado.ZoneStateAuto:
		text = "moving to auto mode"
	//case tado.ZoneStateManual:
	//	// Not implemented: GetNextState never returns ZoneStateManual
	//	// text = "setting to manual temperature control"
	default:
		panic("not implemented")
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
	name, ok := mgr.cache.GetZoneName(zoneID)
	if ok == false {
		name = "unknown"
	}
	return []slack.Attachment{{
		Color: "good",
		Title: "resetting rule for " + name,
		Text:  reason,
	}}
}
