package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Task interface {
	Run(ctx context.Context) error
}

func Schedule(ctx context.Context, task Task, waitTime time.Duration) *Job {
	ctx2, cancel := context.WithCancel(ctx)
	j := &Job{
		task:   task,
		state:  stateUnknown,
		cancel: cancel,
	}
	go j.run(ctx2, waitTime)

	return j
}

type Job struct {
	task   Task
	state  state
	cancel context.CancelFunc
	err    error
	lock   sync.RWMutex
}

func (j *Job) run(ctx context.Context, waitTime time.Duration) {
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
	switch result {
	case stateCompleted, stateFailed:
		completed = true
		j.cancel()
	case stateCanceled:
		err = errors.New("job canceled")
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
