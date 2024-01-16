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

var _ scheduler.Runnable = &Task{}

func newTask(ctx context.Context, api action.TadoSetter, next action.Action, notification chan struct{}) *Task {
	task := Task{
		api:    api,
		action: next,
	}
	task.job = scheduler.NewWithNotification(ctx, &task, notification)
	go task.job.Run(next.Delay)
	return &task
}

func (t Task) Run(ctx context.Context) error {
	return t.action.State.Do(ctx, t.api)
}

func (t Task) scheduledBefore(next action.Action) bool {
	scheduled := t.job.Due().Round(time.Second)
	newJob := time.Now().Add(next.Delay).Round(time.Second)
	return scheduled.Before(newJob) || scheduled.Equal(newJob)
}

func (t Task) Report() string {
	result := t.action.String() + " in " + time.Until(t.job.Due()).Round(time.Second).String()
	if t.action.Label != "" {
		result = t.action.Label + ": " + result
	}
	return result
}
