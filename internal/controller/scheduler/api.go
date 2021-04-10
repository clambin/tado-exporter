package scheduler

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"time"
)

type API interface {
	ScheduleTask(zoneID int, state model.ZoneState, when time.Duration)
	ScheduledState(zoneID int) model.ZoneState
	Stop()
	Run()
}

func (scheduler *Scheduler) ScheduleTask(zoneID int, state model.ZoneState, when time.Duration) {
	scheduler.Schedule <- &Task{
		ZoneID: zoneID,
		State:  state,
		When:   when,
	}
}

func (scheduler *Scheduler) ScheduledState(zoneID int) (state model.ZoneState) {
	response := make(chan model.ZoneState)
	scheduler.Scheduled <- ScheduledRequest{
		ZoneID:   zoneID,
		Response: response,
	}
	return <-response
}

func (scheduler *Scheduler) Stop() {
	scheduler.Cancel <- struct{}{}
}
