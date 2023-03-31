package zonemanager

import (
	"context"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"time"
)

type Task struct {
	api       rules.TadoSetter
	nextState rules.TargetState
	when      time.Time
	job       *scheduler.Job
}

var _ scheduler.Task = &Task{}

func newTask(api rules.TadoSetter, next rules.TargetState) *Task {
	return &Task{
		api:       api,
		nextState: next,
		when:      time.Now().Add(next.Delay),
	}
}

func (j *Task) Run(ctx context.Context) (err error) {
	return j.nextState.State.Do(ctx, j.api, j.nextState.ZoneID)
}

func (j *Task) firesNoLaterThan(delay time.Duration) bool {
	scheduled := int64(time.Until(j.when).Seconds())
	newJob := int64(delay.Seconds())
	return scheduled <= newJob
}

func (j *Task) Report() string {
	return j.nextState.ZoneName + ": " + j.nextState.State.String() + " in " + time.Until(j.when).Round(time.Second).String()
}
