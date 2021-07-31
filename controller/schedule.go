package controller

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/scheduler"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"time"
)

func (controller *Controller) scheduleZoneStateChange(ctx context.Context, zoneID int, state tado.ZoneState, when time.Duration, reason string) {
	// when does this task run?
	activation := time.Now().Add(when)

	// check if we already have a task running for the zoneID
	running, ok := controller.scheduler.GetScheduled(scheduler.TaskID(zoneID))

	if ok {
		// if that change is already pending for that room, and the scheduled change will start earlier, don't schedule the new change
		if state == running.Args[1].(tado.ZoneState) && activation.After(running.Activation) { // running.Sub(activation).Round(time.Minute) <= 0 {
			// log.WithFields(log.Fields{"zone": zoneID, "running": running, "new": activation}).Debug("earlier task already found. won't schedule this task")
			return
		}

		// cancel the running task
		controller.scheduler.Cancel(scheduler.TaskID(zoneID))
		log.WithField("zone", zoneID).Debug("canceled running task")
	}

	controller.scheduler.Schedule(ctx, &scheduler.Task{
		ID:  scheduler.TaskID(zoneID),
		Run: controller.runTask,
		Args: []interface{}{
			zoneID,
			state,
			reason,
		},
		When: when,
	})

	// log.WithFields(log.Fields{"zone": zoneID, "state": state, "when": when.String()}).Debug("scheduled task")

	// post Slack notification, if it will be executed later
	if when > 0 && controller.PostChannel != nil {
		controller.PostChannel <- controller.getNotification(state, reason, activation)
	}
}

func (controller *Controller) cancelZoneStateChange(zoneID int, reason string) {
	if controller.scheduler.Cancel(scheduler.TaskID(zoneID)) == true && controller.PostChannel != nil {
		controller.PostChannel <- controller.getCancelNotification(zoneID, reason)
	}
}

func (controller *Controller) runTask(ctx context.Context, args []interface{}) {
	zoneID := args[0].(int)
	state := args[1].(tado.ZoneState)
	reason := args[2].(string)

	var err error
	switch state {
	case tado.ZoneStateOff:
		err = controller.API.SetZoneOverlay(ctx, zoneID, 5.0)
	case tado.ZoneStateAuto:
		err = controller.API.DeleteZoneOverlay(ctx, zoneID)
	//case tado.ZoneStateManual:
	//	not implemented: GetNextState never returns ZoneStateManual
	//	err = controller.API.SetZoneOverlay(ctx, zoneID, 15.0 /* TODO: state.Temperature.Celsius */)
	default:
		panic("not implemented")
	}

	if err != nil {
		log.WithField("err", err).Warning("unable to update zone")
		return
	}

	if controller.PostChannel != nil {
		controller.PostChannel <- controller.getNotification(state, reason, time.Time{})
	}

	log.WithFields(log.Fields{"zone": zoneID, "state": state}).Debug("executed task")
}

func (controller *Controller) getNotification(state tado.ZoneState, reason string, activation time.Time) (attachment []slack.Attachment) {
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

func (controller *Controller) getCancelNotification(zoneID int, reason string) []slack.Attachment {
	name, ok := controller.cache.GetZoneName(zoneID)
	if ok == false {
		name = "unknown"
	}
	return []slack.Attachment{{
		Color: "good",
		Title: "resetting rule for " + name,
		Text:  reason,
	}}
}