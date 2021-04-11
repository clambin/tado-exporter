package scheduler

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/slack-go/slack"
	"time"
)

type API interface {
	Run()
	Stop()
	ScheduleTask(zoneID int, state models.ZoneState, when time.Duration)
	UnscheduleTask(zoneID int)
	ScheduledState(zoneID int) models.ZoneState
	ReportTasks(_ ...string) []slack.Attachment
}

func (scheduler *Scheduler) ScheduleTask(zoneID int, state models.ZoneState, when time.Duration) {
	scheduler.Schedule <- &Task{
		ZoneID: zoneID,
		State:  state,
		When:   when,
	}
}

func (scheduler *Scheduler) UnscheduleTask(zoneID int) {
	scheduler.Unschedule <- zoneID
}

func (scheduler *Scheduler) ScheduledState(zoneID int) (state models.ZoneState) {
	response := make(chan models.ZoneState)
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
