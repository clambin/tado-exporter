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

type Job struct {
	when   time.Time
	runner Runnable
	err    error
	cancel context.CancelFunc
	done   chan struct{}
	state  State
	lock   sync.RWMutex
}

type Runnable interface {
	Run(context.Context) error
}

type RunFunc func(context.Context) error

func (r RunFunc) Run(ctx context.Context) error {
	return r(ctx)
}

func Schedule(ctx context.Context, runner Runnable, delay time.Duration, ch chan struct{}) *Job {
	subCtx, cancel := context.WithCancel(ctx)
	j := Job{
		runner: runner,
		state:  StateUnknown,
		cancel: cancel,
		done:   ch,
	}
	go j.Run(subCtx, delay)
	return &j
}

func (j *Job) Run(ctx context.Context, waitTime time.Duration) {
	j.schedule(waitTime)
	select {
	case <-ctx.Done():
		j.markCanceled()
	case <-time.After(waitTime):
		err := j.runner.Run(ctx)
		j.markCompleted(err)
	}
	j.Cancel()
	if j.done != nil {
		j.done <- struct{}{}
	}
}

func (j *Job) Cancel() {
	if j.cancel != nil {
		j.cancel()
	}
}

func (j *Job) Result() (bool, error) {
	state, err, _ := j.GetState()
	return state == StateCompleted || state == StateCanceled, err
}

func (j *Job) schedule(waitTime time.Duration) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = StateScheduled
	j.when = time.Now().Add(waitTime)
}

func (j *Job) markCompleted(err error) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = StateCompleted
	j.err = err
}

func (j *Job) markCanceled() {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = StateCanceled
	j.err = ErrCanceled
}

func (j *Job) GetState() (State, error, time.Time) {
	j.lock.RLock()
	defer j.lock.RUnlock()
	return j.state, j.err, j.when
}

func (j *Job) Due() time.Time {
	if s, _, when := j.GetState(); s == StateScheduled {
		return when
	}
	return time.Time{}
}
