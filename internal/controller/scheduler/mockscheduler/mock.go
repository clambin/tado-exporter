package mockscheduler

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/slack-go/slack"
	"sync"
	"time"
)

type Task struct {
	State  model.ZoneState
	Expiry time.Time
}

type MockScheduler struct {
	tasks map[int]Task
	lock  sync.Mutex
}

func New() *MockScheduler {
	return &MockScheduler{tasks: make(map[int]Task)}
}

func (scheduler *MockScheduler) ScheduleTask(zoneID int, state model.ZoneState, when time.Duration) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	expiry := time.Now().Add(when)
	if task, ok := scheduler.tasks[zoneID]; ok {
		if expiry.Before(task.Expiry) {
			scheduler.tasks[zoneID] = Task{
				State:  state,
				Expiry: expiry,
			}
		}
	} else {
		scheduler.tasks[zoneID] = Task{
			State:  state,
			Expiry: expiry,
		}
	}
}

func (scheduler *MockScheduler) ScheduledState(zoneID int) (state model.ZoneState) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	if task, ok := scheduler.tasks[zoneID]; ok {
		if task.Expiry.After(time.Now()) {
			state = task.State
		} else {
			state = model.ZoneState{State: model.Unknown}
			delete(scheduler.tasks, zoneID)
		}
	} else {
		state = model.ZoneState{State: model.Unknown}
	}
	return
}

func (scheduler *MockScheduler) Run() {
}

func (scheduler *MockScheduler) Stop() {
}

func (scheduler *MockScheduler) ReportTasks(_ ...string) []slack.Attachment {
	return []slack.Attachment{}
}
