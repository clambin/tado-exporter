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
	When       time.Duration
	Activation time.Time
	fire       chan TaskID
	cancel     context.CancelFunc
	Args       []interface{}
}

func (task Task) wait(ctx context.Context) {
	timer := time.NewTimer(task.When)
	defer timer.Stop()

	waiting := true
	for waiting {
		select {
		case <-ctx.Done():
			waiting = false
		case <-timer.C:
			task.fire <- task.ID
			waiting = false
		}
	}
}
