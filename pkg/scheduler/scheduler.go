package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCanceled = errors.New("job canceled")

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
	ctx2, cancel := context.WithCancel(ctx)
	j := Job{
		task:   task,
		state:  stateUnknown,
		Cancel: cancel,
	}
	go j.run(ctx2, waitTime)

	return &j
}

func (j *Job) run(ctx context.Context, waitTime time.Duration) {
	j.setState(stateScheduled, nil)
	select {
	case <-ctx.Done():
		j.setState(stateCanceled, ErrCanceled)
	case <-time.After(waitTime):
		err := j.task.Run(ctx)
		j.setState(stateCompleted, err)
	}
	j.Cancel()
}

func (j *Job) Result() (bool, error) {
	result, err := j.getState()
	completed := result == stateCompleted || result == stateCanceled
	return completed, err
}

func (j *Job) setState(state state, err error) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = state
	j.err = err
}

func (j *Job) getState() (state, error) {
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
