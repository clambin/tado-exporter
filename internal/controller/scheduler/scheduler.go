package scheduler

import (
	"context"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Scheduler struct {
	tasks map[TaskID]Task
	fire  chan TaskID
	lock  sync.RWMutex
}

func New() *Scheduler {
	return &Scheduler{
		tasks: make(map[TaskID]Task),
		fire:  make(chan TaskID),
	}
}

func (scheduler *Scheduler) Run(ctx context.Context) {
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case id := <-scheduler.fire:
			scheduler.run(ctx, id)
		}
	}
	scheduler.CancelAll()
}

func (scheduler *Scheduler) Schedule(ctx context.Context, task Task) {
	scheduler.lock.RLock()
	defer scheduler.lock.RUnlock()

	if oldTask, found := scheduler.tasks[task.ID]; found {
		oldTask.cancel()
	}

	newCtx, cancel := context.WithCancel(ctx)

	task.cancel = cancel
	task.fire = scheduler.fire

	scheduler.tasks[task.ID] = task
	go task.wait(newCtx)
}

func (scheduler *Scheduler) Cancel(taskID TaskID) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	if task, found := scheduler.tasks[taskID]; found {
		task.cancel()
		delete(scheduler.tasks, taskID)
	}
}

func (scheduler *Scheduler) CancelAll() {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	for _, task := range scheduler.tasks {
		task.cancel()
	}
	scheduler.tasks = make(map[TaskID]Task)
}

func (scheduler *Scheduler) GetScheduled(id TaskID) (task Task, found bool) {
	scheduler.lock.RLock()
	defer scheduler.lock.RUnlock()

	task, found = scheduler.tasks[id]
	return
}

func (scheduler *Scheduler) GetAllScheduled() (tasks []Task) {
	scheduler.lock.RLock()
	defer scheduler.lock.RUnlock()

	for _, task := range scheduler.tasks {
		tasks = append(tasks, task)
	}
	return
}

func (scheduler *Scheduler) run(ctx context.Context, taskID TaskID) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	if task, found := scheduler.tasks[taskID]; found {
		task.Run(ctx, task.Args)
		delete(scheduler.tasks, taskID)
	} else {
		log.WithField("taskID", taskID).Warning("Task no longer running. ignoring")
	}
}
