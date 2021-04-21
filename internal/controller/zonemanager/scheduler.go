package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	log "github.com/sirupsen/logrus"
	"time"
)

func (mgr *Manager) scheduleTask(zoneID int, state models.ZoneState, when time.Duration, reason string) {
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
		Cancel:     make(chan struct{}),
		zoneID:     zoneID,
		zoneName:   mgr.getZoneName(zoneID),
		state:      state,
		reason:     reason,
		activation: time.Now().Add(when),
	}
	if when == 0 {
		// run the task directly
		mgr.runTask(task)
	} else {
		// schedule it
		mgr.tasks[zoneID] = task
		go task.Run(when, mgr.fire)

		log.WithFields(log.Fields{
			"zone":  zoneID,
			"state": state.String(),
			"when":  when.String(),
		}).Debug("scheduled task")

		// post slack notification
		if mgr.postChannel != nil {
			mgr.postChannel <- task.notification()
		}
	}
}

func (mgr *Manager) unscheduleTask(zoneID int, reason string) {
	if task, ok := mgr.tasks[zoneID]; ok == true {
		task.Cancel <- struct{}{}

		if mgr.postChannel != nil {
			mgr.postChannel <- task.cancelNotification(reason)
		}

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
			task.activation = time.Time{}
			mgr.postChannel <- task.notification()
		}
		log.WithFields(log.Fields{"zone": task.zoneID, "state": task.state.String()}).Debug("executed task")
	} else {
		log.WithField("err", err).Warning("unable to update zone")
	}

	// unregister the completed task
	delete(mgr.tasks, task.zoneID)
}
