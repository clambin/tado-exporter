package processor

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"time"
)

type Task struct {
	api    action.TadoSetter
	action action.Action
	job    *scheduler.Job
}

var _ scheduler.Task = &Task{}

func newTask(ctx context.Context, api action.TadoSetter, next action.Action, notification chan struct{}) *Task {
	task := Task{
		api:    api,
		action: next,
	}
	task.job = scheduler.ScheduleWithNotification(ctx, &task, next.Delay, notification)
	return &task
}

func (t *Task) Run(ctx context.Context) (err error) {
	return t.action.State.Do(ctx, t.api)
}

func (t *Task) firesNoLaterThan(next action.Action) bool {
	scheduled := t.job.TimeToFire().Round(time.Second)
	newJob := next.Delay.Round(time.Second)
	return scheduled <= newJob
}

func (t *Task) Report() string {
	result := t.action.String() + " in " + t.job.TimeToFire().Round(time.Second).String()
	if t.action.Label != "" {
		result = t.action.Label + ": " + result
	}
	return result
}
