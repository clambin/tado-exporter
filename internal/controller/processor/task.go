package processor

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"time"
)

type Task struct {
	client action.TadoClient
	action action.Action
	job    *scheduler.Job
}

var _ scheduler.Runnable = &Task{}

func scheduleTask(ctx context.Context, client action.TadoClient, a action.Action, done chan struct{}) *Task {
	task := Task{
		client: client,
		action: a,
	}
	task.job = scheduler.Schedule(ctx, &task, a.Delay, done)
	return &task
}

func (t Task) Run(ctx context.Context) error {
	return t.action.State.Do(ctx, t.client)
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
