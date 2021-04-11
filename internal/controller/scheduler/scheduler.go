package scheduler

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

type Scheduler struct {
	API         tado.API
	Cancel      chan struct{}
	Schedule    chan *Task
	Unschedule  chan int
	Scheduled   chan ScheduledRequest
	Report      chan struct{}
	fire        chan *Task
	tasks       map[int]*scheduledTask
	postChannel slackbot.PostChannel
	nameCache   map[int]string
}

type Task struct {
	ZoneID int
	State  models.ZoneState
	When   time.Duration
}

type ScheduledRequest struct {
	ZoneID   int
	Response chan models.ZoneState
}

func New(API tado.API, postChannel slackbot.PostChannel) API {
	return &Scheduler{
		API:         API,
		Cancel:      make(chan struct{}),
		Schedule:    make(chan *Task),
		Unschedule:  make(chan int),
		Scheduled:   make(chan ScheduledRequest),
		Report:      make(chan struct{}),
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
		case task := <-scheduler.Schedule:
			scheduler.schedule(task)
		case zoneID := <-scheduler.Unschedule:
			scheduler.unschedule(zoneID)
		case task := <-scheduler.fire:
			scheduler.runTask(task)
		case req := <-scheduler.Scheduled:
			req.Response <- scheduler.getScheduledState(req.ZoneID)
		case <-scheduler.Report:
			if scheduler.postChannel != nil {
				scheduler.postChannel <- scheduler.reportTasks()
			}
		}
	}
	close(scheduler.Cancel)
}

func (scheduler *Scheduler) schedule(task *Task) {
	// check if we already have a task running for the zoneID
	running, ok := scheduler.tasks[task.ZoneID]

	// if we're already setting that state, ignore the new task, unless it sets that state earlier
	if ok && running.task.Equals(task) && running.activation.Before(time.Now().Add(task.When)) {
		return
	}

	// cancel a previously running task
	if ok == true {
		running.Cancel <- struct{}{}
		log.WithField("zone", task.ZoneID).Debug("canceled running task")
	}

	if task.When == 0 {
		// if the task is immediate, do it here
		scheduler.runTask(task)
	} else {
		// schedule a new task
		s := &scheduledTask{
			Cancel:      make(chan struct{}),
			task:        task,
			timer:       time.NewTimer(task.When),
			activation:  time.Now().Add(task.When),
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
}

func (scheduler *Scheduler) unschedule(zoneID int) {
	if task, ok := scheduler.tasks[zoneID]; ok == true {
		task.Cancel <- struct{}{}
		delete(scheduler.tasks, zoneID)
	}
}

func (scheduler *Scheduler) runTask(task *Task) {
	var err error
	switch task.State.State {
	case models.ZoneOff:
		err = scheduler.API.SetZoneOverlay(task.ZoneID, 5.0)
	case models.ZoneAuto:
		err = scheduler.API.DeleteZoneOverlay(task.ZoneID)
	case models.ZoneManual:
		err = scheduler.API.SetZoneOverlay(task.ZoneID, task.State.Temperature.Celsius)
	}
	if err == nil {
		if scheduler.postChannel != nil {
			scheduler.postChannel <- scheduler.notifyExecutedTask(task)
		}
		log.WithField("zone", task.ZoneID).Debug("executed task")
	} else {
		log.WithField("err", err).Warning("unable to update zone")
	}

	// unregister the completed task
	delete(scheduler.tasks, task.ZoneID)
}

func (scheduler *Scheduler) getScheduledState(zoneID int) (state models.ZoneState) {
	if scheduled, ok := scheduler.tasks[zoneID]; ok == true {
		state = scheduled.task.State
	} else {
		state = models.ZoneState{
			State: models.ZoneUnknown,
		}
	}
	return
}

func (a *Task) Equals(b *Task) bool {
	return a.ZoneID == b.ZoneID && a.State.Equals(b.State)
}
