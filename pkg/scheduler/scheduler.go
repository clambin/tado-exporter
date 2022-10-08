package scheduler

import (
	"context"
	"sync"
	"time"
)

func Schedule(ctx context.Context, task Task, waitTime time.Duration) *Job {
	ctx2, cancel := context.WithCancel(ctx)
	j := &Job{
		task:   task,
		state:  stateUnknown,
		cancel: cancel,
	}
	go j.Run(ctx2, waitTime)

	return j
}

type Task interface {
	Run(ctx context.Context) error
}

type Job struct {
	task   Task
	state  state
	cancel context.CancelFunc
	err    error
	lock   sync.RWMutex
}

func (j *Job) Run(ctx context.Context, waitTime time.Duration) {
	j.setState(stateScheduled, nil)
	select {
	case <-ctx.Done():
		return
	case <-time.After(waitTime):
		err := j.task.Run(ctx)
		s := stateCompleted
		if err != nil {
			s = stateFailed
		}
		j.setState(s, err)
	}
}

func (j *Job) Cancel() {
	j.cancel()
	j.setState(stateCanceled, nil)
}

func (j *Job) Result() (completed bool, err error) {
	var result state
	result, err = j.getState()
	if completed = result.done(); completed {
		j.cancel()
	}
	return
}

func (j *Job) setState(state state, err error) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = state
	j.err = err
}

func (j *Job) getState() (state state, err error) {
	j.lock.RLock()
	defer j.lock.RUnlock()
	return j.state, j.err
}

type state int

const (
	stateUnknown state = iota
	stateScheduled
	stateCanceled
	stateCompleted
	stateFailed
)

func (s state) done() bool {
	return s == stateCompleted || s == stateFailed || s == stateCanceled
}
