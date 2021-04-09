package scheduler

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

type Scheduler struct {
	API         tado.API
	Cancel      chan struct{}
	Register    chan *Task
	fire        chan *Task
	tasks       map[int]*scheduledTask
	postChannel slackbot.PostChannel
	nameCache   map[int]string
}

type Task struct {
	ZoneID int
	State  model.ZoneState
	When   time.Duration
}

func New(API tado.API, postChannel slackbot.PostChannel) *Scheduler {
	return &Scheduler{
		API:         API,
		Cancel:      make(chan struct{}),
		Register:    make(chan *Task),
		fire:        make(chan *Task),
		tasks:       make(map[int]*scheduledTask),
		postChannel: postChannel,
		nameCache:   make(map[int]string),
	}
}

func (scheduler *Scheduler) Run() {
loop:
	for {
		select {
		case <-scheduler.Cancel:
			break loop
		case task := <-scheduler.Register:
			scheduler.schedule(task)
		case task := <-scheduler.fire:
			scheduler.runTask(task)
		}
	}
	close(scheduler.Cancel)
}

func (scheduler *Scheduler) schedule(task *Task) {
	// cancel a previously running task
	if running, ok := scheduler.tasks[task.ZoneID]; ok == true {
		running.Cancel <- struct{}{}
		log.WithField("zone", task.ZoneID).Debug("canceled running task")
	}
	// run a new task
	s := &scheduledTask{
		Cancel:      make(chan struct{}),
		task:        task,
		timer:       time.NewTimer(task.When),
		fireChannel: scheduler.fire,
	}
	scheduler.tasks[task.ZoneID] = s
	go s.Run()

	log.WithFields(log.Fields{
		"zone":  task.ZoneID,
		"state": task.State.String(),
		"when":  task.When.String(),
	}).Debug("scheduled task")

	if scheduler.postChannel != nil {
		scheduler.postChannel <- scheduler.notifyPendingTask(task)
	}
}

func (scheduler *Scheduler) runTask(task *Task) {
	var err error
	switch task.State.State {
	case model.Off:
		err = scheduler.API.SetZoneOverlay(task.ZoneID, 5.0)
	case model.Auto:
		err = scheduler.API.DeleteZoneOverlay(task.ZoneID)
	case model.Manual:
		err = scheduler.API.SetZoneOverlay(task.ZoneID, task.State.Temperature.Celsius)
	}
	if err == nil {
		if scheduler.postChannel != nil {
			scheduler.postChannel <- scheduler.notifyExecutedTask(task)
		}
	} else {
		log.WithField("err", err).Debug("unable to update zone")
	}

	// unregister the completed task
	delete(scheduler.tasks, task.ZoneID)

	log.WithField("zone", task.ZoneID).Debug("executed task")
}
