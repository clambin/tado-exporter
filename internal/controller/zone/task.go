package zone

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"time"
)

type Task struct {
	api       rules.TadoSetter
	nextState rules.Action
	job       *scheduler.Job
}

var _ scheduler.Task = &Task{}

func newTask(ctx context.Context, api rules.TadoSetter, next rules.Action, notification chan struct{}) *Task {
	task := Task{
		api:       api,
		nextState: next,
	}
	task.job = scheduler.ScheduleWithNotification(ctx, &task, next.Delay, notification)
	return &task
}

func (t *Task) Run(ctx context.Context) (err error) {
	return t.nextState.State.Do(ctx, t.api, t.nextState.ZoneID)
}

func (t *Task) firesNoLaterThan(next rules.Action) bool {
	scheduled := t.job.TimeToFire().Round(time.Second)
	newJob := next.Delay.Round(time.Second)
	return scheduled <= newJob
}

func (t *Task) Report() string {
	return t.nextState.ZoneName + ": " + t.nextState.State.Action() + " in " + t.job.TimeToFire().Round(time.Second).String()
}
