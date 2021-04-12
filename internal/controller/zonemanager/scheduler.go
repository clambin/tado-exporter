package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	log "github.com/sirupsen/logrus"
	"time"
)

func (mgr *Manager) scheduleTask(zoneID int, state models.ZoneState, when time.Duration) {
	// check if we already have a task running for the zoneID
	if running, ok := mgr.tasks[zoneID]; ok {
		// if we're already setting that state, ignore the new task, unless it sets that state earlier
		if running.state.Equals(state) && running.activation.Before(time.Now().Add(when)) {
			return
		}

		running.Cancel <- struct{}{}
		log.WithField("zone", zoneID).Debug("canceled running task")
	}

	// create the new task
	task := &Task{
		Cancel:      make(chan struct{}),
		zoneID:      zoneID,
		state:       state,
		timer:       time.NewTimer(when),
		activation:  time.Now().Add(when),
		fireChannel: mgr.fire,
	}
	if when == 0 {
		// if the task is immediate, do it here
		mgr.runTask(task)
	} else {
		// otherwise, schedule it
		mgr.tasks[zoneID] = task
		go task.Run()

		log.WithFields(log.Fields{
			"zone":  zoneID,
			"state": state.String(),
			"when":  when.String(),
		}).Debug("scheduled task")

		if mgr.postChannel != nil {
			mgr.postChannel <- mgr.notifyPendingTask(task)
		}
	}
}

func (mgr *Manager) unscheduleTask(zoneID int) {
	if task, ok := mgr.tasks[zoneID]; ok == true {
		task.Cancel <- struct{}{}
		delete(mgr.tasks, zoneID)
	}
}

func (mgr *Manager) runTask(task *Task) {
	var err error
	switch task.state.State {
	case models.ZoneOff:
		err = mgr.API.SetZoneOverlay(task.zoneID, 5.0)
	case models.ZoneAuto:
		err = mgr.API.DeleteZoneOverlay(task.zoneID)
	case models.ZoneManual:
		err = mgr.API.SetZoneOverlay(task.zoneID, task.state.Temperature.Celsius)
	}
	if err == nil {
		if mgr.postChannel != nil {
			mgr.postChannel <- mgr.notifyExecutedTask(task)
		}
		log.WithFields(log.Fields{"zone": task.zoneID, "state": task.state.String()}).Debug("executed task")
	} else {
		log.WithField("err", err).Warning("unable to update zone")
	}

	// unregister the completed task
	delete(mgr.tasks, task.zoneID)
}
