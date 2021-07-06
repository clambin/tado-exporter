package scheduler

import (
	"context"
	"time"
)

type TaskID int
type TaskFunc func(ctx context.Context, args []interface{})

type Task struct {
	ID         TaskID
	Run        TaskFunc
	Args       []interface{}
	When       time.Duration
	Activation time.Time
	fire       chan TaskID
	cancel     context.CancelFunc
}

func (task Task) wait(ctx context.Context) {
	timer := time.NewTimer(task.When)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			task.fire <- task.ID
		}
	}
}
