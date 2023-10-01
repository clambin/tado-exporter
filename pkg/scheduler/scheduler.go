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
	j.setState(StateScheduled, nil)
	j.when = time.Now().Add(waitTime)
	select {
	case <-ctx.Done():
		j.setState(StateCanceled, ErrCanceled)
	case <-time.After(waitTime):
		err := j.task.Run(ctx)
		j.setState(StateCompleted, err)
	}
	j.Cancel()
	if j.notify != nil {
		j.notify <- struct{}{}
	}
}

func (j *Job) Result() (bool, error) {
	result, err := j.GetState()
	completed := result == StateCompleted || result == StateCanceled
	return completed, err
}

func (j *Job) setState(state State, err error) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = state
	j.err = err
}

func (j *Job) GetState() (State, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()
	return j.state, j.err
}

func (j *Job) TimeToFire() time.Duration {
	s, err := j.GetState()
	if err != nil || s != StateScheduled {
		return 0
	}
	return time.Until(j.when)
}
