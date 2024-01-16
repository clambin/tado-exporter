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
	when       time.Time
	runner     Runnable
	err        error
	subCtx     context.Context
	cancelFunc context.CancelFunc
	state      State
	notify     chan struct{}
	lock       sync.RWMutex
}

type Runnable interface {
	Run(context.Context) error
}

func New(ctx context.Context, runner Runnable) *Job {
	return NewWithNotification(ctx, runner, nil)
}

func NewWithNotification(ctx context.Context, runner Runnable, ch chan struct{}) *Job {
	subCtx, cancel := context.WithCancel(ctx)
	return &Job{
		runner:     runner,
		state:      StateUnknown,
		subCtx:     subCtx,
		cancelFunc: cancel,
		notify:     ch,
	}
}

func (j *Job) Run(waitTime time.Duration) {
	j.setScheduled(waitTime)
	select {
	case <-j.subCtx.Done():
		j.setCanceled()
	case <-time.After(waitTime):
		j.setCompleted(j.runner.Run(j.subCtx))
	}
	j.Cancel()
	if j.notify != nil {
		j.notify <- struct{}{}
	}
}

func (j *Job) Cancel() {
	j.cancelFunc()
}

func (j *Job) Result() (bool, error) {
	state, err, _ := j.GetState()
	return state == StateCompleted || state == StateCanceled, err
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
