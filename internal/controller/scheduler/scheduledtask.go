package scheduler

import (
	"time"
)

type scheduledTask struct {
	Cancel      chan struct{}
	task        *Task
	timer       *time.Timer
	activation  time.Time
	fireChannel chan *Task
}

func (task *scheduledTask) Run() {
loop:
	for {
		select {
		case <-task.Cancel:
			break loop
		case <-task.timer.C:
			task.fireChannel <- task.task
			break loop
		}
	}
	task.timer.Stop()
}
