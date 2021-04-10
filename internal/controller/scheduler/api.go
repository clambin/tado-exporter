package scheduler

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/slack-go/slack"
	"time"
)

type API interface {
	Run()
	Stop()
	ScheduleTask(zoneID int, state model.ZoneState, when time.Duration)
	ScheduledState(zoneID int) model.ZoneState
	ReportTasks(_ ...string) []slack.Attachment
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

func (scheduler *Scheduler) ReportTasks(_ ...string) []slack.Attachment {
	scheduler.Report <- struct{}{}
	return []slack.Attachment{}
}
