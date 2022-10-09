package scheduler

import (
	"context"
	"sync"
	"time"
)

type Task interface {
	Run(ctx context.Context) error
}

type Job struct {
	Cancel context.CancelFunc
	task   Task
	state  state
	err    error
	lock   sync.RWMutex
}

func Schedule(ctx context.Context, task Task, waitTime time.Duration) *Job {
	j := &Job{
		task:  task,
		state: stateUnknown,
	}
	var ctx2 context.Context
	ctx2, j.Cancel = context.WithCancel(ctx)
	go j.run(ctx2, waitTime)

	return j
}

func (j *Job) run(ctx context.Context, waitTime time.Duration) {
	j.setState(stateScheduled, nil)
	select {
	case <-ctx.Done():
		j.setState(stateCanceled, ErrCanceled)
		return
	case <-time.After(waitTime):
		err := j.task.Run(ctx)
		if err != nil {
			err = &errFailed{err: err}
		}
		j.Cancel()
		j.setState(stateCompleted, err)
	}
}

func (j *Job) Result() (completed bool, err error) {
	var result state
	result, err = j.getState()
	completed = result == stateCompleted || result == stateCanceled
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
)
