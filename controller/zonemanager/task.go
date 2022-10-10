package zonemanager

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	log "github.com/sirupsen/logrus"
	"time"
)

type Task struct {
	api       tado.API
	nextState *rules.NextState
	when      time.Time
	job       *scheduler.Job
}

var _ scheduler.Task = &Task{}

func newTask(api tado.API, next *rules.NextState) *Task {
	return &Task{
		api:       api,
		nextState: next,
		when:      time.Now().Add(next.Delay),
	}
}

func (j *Task) Run(ctx context.Context) (err error) {
	switch j.nextState.State {
	case tado.ZoneStateAuto:
		err = j.api.DeleteZoneOverlay(ctx, j.nextState.ZoneID)
	case tado.ZoneStateOff:
		err = j.api.SetZoneOverlay(ctx, j.nextState.ZoneID, 5.0)
	default:
		err = fmt.Errorf("invalid queued state for zone '%s': %d", j.nextState.ZoneName, j.nextState.State)
	}
	return
}

func (j *Task) firesNoLaterThan(delay time.Duration) bool {
	scheduled := int64(time.Until(j.when).Seconds())
	newJob := int64(delay.Seconds())

	log.Debugf("scheduled job: %d, new job: %d", scheduled, newJob)

	return scheduled <= newJob
}

func (j *Task) Report() string {
	var action string
	switch j.nextState.State {
	case tado.ZoneStateOff:
		action = "switching off heating"
	case tado.ZoneStateAuto:
		action = "moving to auto mode"
	}

	return j.nextState.ZoneName + ": " + action + " in " + time.Until(j.when).Round(time.Second).String()
}
