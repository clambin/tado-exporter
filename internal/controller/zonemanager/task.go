package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"time"
)

type Task struct {
	zoneID      int
	state       models.ZoneState
	timer       *time.Timer
	activation  time.Time
	Cancel      chan struct{}
	fireChannel chan *Task
}

func (task *Task) Run() {
loop:
	for {
		select {
		case <-task.Cancel:
			break loop
		case <-task.timer.C:
			task.fireChannel <- task
			break loop
		}
	}
	task.timer.Stop()
}
