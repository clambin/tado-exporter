package scheduler

import (
	"context"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Scheduler struct {
	tasks map[TaskID]*Task
	fire  chan TaskID
	lock  sync.RWMutex
}

func (scheduler *Scheduler) Run(ctx context.Context) {
	log.Info("scheduler started")
	scheduler.init()

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case id := <-scheduler.fire:
			scheduler.run(ctx, id)
		}
	}
	scheduler.CancelAll()
	log.Info("scheduler stopped")
}

func (scheduler *Scheduler) init() {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	if scheduler.tasks == nil {
		scheduler.tasks = make(map[TaskID]*Task)
	}
	if scheduler.fire == nil {
		scheduler.fire = make(chan TaskID)
	}
}

func (scheduler *Scheduler) Schedule(ctx context.Context, task *Task) {
	scheduler.init()
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	if oldTask, found := scheduler.tasks[task.ID]; found {
		oldTask.cancel()
	}

	newCtx, cancel := context.WithCancel(ctx)

	task.cancel = cancel
	task.fire = scheduler.fire
	task.Activation = time.Now().Add(task.When)

	scheduler.tasks[task.ID] = task
	go task.wait(newCtx)
}

func (scheduler *Scheduler) Cancel(taskID TaskID) (found bool) {
	scheduler.init()
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	var task *Task
	if task, found = scheduler.tasks[taskID]; found {
		task.cancel()
		delete(scheduler.tasks, taskID)
	}
	return
}

func (scheduler *Scheduler) CancelAll() {
	scheduler.init()
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	for _, task := range scheduler.tasks {
		task.cancel()
	}
	scheduler.tasks = make(map[TaskID]*Task)
}

func (scheduler *Scheduler) GetScheduled(id TaskID) (task *Task, found bool) {
	scheduler.init()
	scheduler.lock.RLock()
	defer scheduler.lock.RUnlock()

	task, found = scheduler.tasks[id]
	return
}

func (scheduler *Scheduler) GetAllScheduled() (tasks []*Task) {
	scheduler.init()
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

	task, found := scheduler.tasks[taskID]

	if !found {
		log.WithField("taskID", taskID).Warning("Task no longer running. ignoring")
		return
	}

	log.WithField("taskID", taskID).Debug("task running")
	task.Run(ctx, task.Args)
	delete(scheduler.tasks, taskID)
}
