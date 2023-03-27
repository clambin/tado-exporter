package zonemanager

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/clambin/tado-exporter/poller"
	"time"
)

type Task struct {
	api       TadoSetter
	nextState rules.TargetState
	when      time.Time
	job       *scheduler.Job
}

//go:generate mockery --name TadoSetter
type TadoSetter interface {
	DeleteZoneOverlay(context.Context, int) error
	SetZoneOverlay(context.Context, int, float64) error
}

var _ scheduler.Task = &Task{}

func newTask(api TadoSetter, next rules.TargetState) *Task {
	return &Task{
		api:       api,
		nextState: next,
		when:      time.Now().Add(next.Delay),
	}
}

func (j *Task) Run(ctx context.Context) (err error) {
	switch j.nextState.State {
	case poller.ZoneStateAuto:
		err = j.api.DeleteZoneOverlay(ctx, j.nextState.ZoneID)
	case poller.ZoneStateOff:
		err = j.api.SetZoneOverlay(ctx, j.nextState.ZoneID, 5.0)
	default:
		err = fmt.Errorf("invalid queued state for zone '%s': %d", j.nextState.ZoneName, j.nextState.State)
	}
	return
}

func (j *Task) firesNoLaterThan(delay time.Duration) bool {
	scheduled := int64(time.Until(j.when).Seconds())
	newJob := int64(delay.Seconds())
	return scheduled <= newJob
}

func (j *Task) Report() string {
	var action string
	switch j.nextState.State {
	case poller.ZoneStateOff:
		action = "switching off heating"
	case poller.ZoneStateAuto:
		action = "moving to auto mode"
	}

	return j.nextState.ZoneName + ": " + action + " in " + time.Until(j.when).Round(time.Second).String()
}
