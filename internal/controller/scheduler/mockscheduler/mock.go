package mockscheduler

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/slack-go/slack"
	"sync"
	"time"
)

type Task struct {
	State  models.ZoneState
	Expiry time.Time
}

type MockScheduler struct {
	tasks map[int]Task
	lock  sync.Mutex
}

func New() *MockScheduler {
	return &MockScheduler{tasks: make(map[int]Task)}
}

func (scheduler *MockScheduler) ScheduleTask(zoneID int, state models.ZoneState, when time.Duration) {
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

func (scheduler *MockScheduler) ScheduledState(zoneID int) (state models.ZoneState) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	if task, ok := scheduler.tasks[zoneID]; ok {
		if task.Expiry.After(time.Now()) {
			state = task.State
		} else {
			state = models.ZoneState{State: models.ZoneUnknown}
			delete(scheduler.tasks, zoneID)
		}
	} else {
		state = models.ZoneState{State: models.ZoneUnknown}
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
