package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateUnknown State = iota
	StateScheduled
	StateCanceled
	StateCompleted
)

var ErrCanceled = errors.New("job canceled")

type Task interface {
	Run(ctx context.Context) error
}

type Job struct {
	Cancel context.CancelFunc
	task   Task
	state  State
	err    error
	when   time.Time
	lock   sync.RWMutex
	notify chan struct{}
}

func Schedule(ctx context.Context, task Task, waitTime time.Duration) *Job {
	return ScheduleWithNotification(ctx, task, waitTime, nil)
}

func ScheduleWithNotification(ctx context.Context, task Task, waitTime time.Duration, ch chan struct{}) *Job {
	subCtx, cancel := context.WithCancel(ctx)
	j := Job{
		task:   task,
		state:  StateUnknown,
		Cancel: cancel,
		notify: ch,
	}
	go j.run(subCtx, waitTime)

	return &j
}

func (j *Job) run(ctx context.Context, waitTime time.Duration) {
	j.setScheduled(waitTime)
	select {
	case <-ctx.Done():
		j.setCanceled()
	case <-time.After(waitTime):
		j.setCompleted(j.task.Run(ctx))
	}
	j.Cancel()
	if j.notify != nil {
		j.notify <- struct{}{}
	}
}

func (j *Job) Result() (bool, error) {
	result, err, _ := j.GetState()
	completed := result == StateCompleted || result == StateCanceled
	return completed, err
}

func (j *Job) setScheduled(waitTime time.Duration) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = StateScheduled
	j.when = time.Now().Add(waitTime)
}

func (j *Job) setCompleted(err error) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = StateCompleted
	j.err = err
}

func (j *Job) setCanceled() {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = StateCanceled
	j.err = ErrCanceled
}

func (j *Job) GetState() (State, error, time.Duration) {
	j.lock.RLock()
	defer j.lock.RUnlock()
	return j.state, j.err, time.Until(j.when)
}

func (j *Job) TimeToFire() time.Duration {
	s, _, when := j.GetState()
	if s != StateScheduled {
		return 0
	}
	return when
}
