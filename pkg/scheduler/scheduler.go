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
	when       time.Time
	task       Task
	err        error
	cancelFunc context.CancelFunc
	state      State
	notify     chan struct{}
	lock       sync.RWMutex
}

func (j *Job) Cancel() {
	j.cancelFunc()
}

func Schedule(ctx context.Context, task Task, waitTime time.Duration) *Job {
	return ScheduleWithNotification(ctx, task, waitTime, nil)
}

func ScheduleWithNotification(ctx context.Context, task Task, waitTime time.Duration, ch chan struct{}) *Job {
	subCtx, cancel := context.WithCancel(ctx)
	j := Job{
		task:       task,
		state:      StateUnknown,
		cancelFunc: cancel,
		notify:     ch,
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
