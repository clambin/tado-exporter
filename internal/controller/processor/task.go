package processor

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"time"
)

type Task struct {
	action action.Action
	job    *scheduler.Job
}

func scheduleTask(ctx context.Context, client action.TadoClient, a action.Action, done chan struct{}) *Task {
	f := scheduler.RunFunc(func(ctx context.Context) error {
		return a.State.Do(ctx, client)
	})
	return &Task{
		action: a,
		job:    scheduler.Schedule(ctx, f, a.Delay, done),
	}
}

func (t Task) scheduledBefore(a action.Action) bool {
	currentActionDue := t.job.Due().Round(time.Second)
	nextActionDue := time.Now().Add(a.Delay).Round(time.Second)
	return currentActionDue.Before(nextActionDue) || currentActionDue.Equal(nextActionDue)
}

func (t Task) Report() string {
	result := t.action.String() + " in " + time.Until(t.job.Due()).Round(time.Second).String()
	if t.action.Label != "" {
		result = t.action.Label + ": " + result
	}
	return result
}
