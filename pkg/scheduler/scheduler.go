// Package scheduler provides a basic mechanism to run a job after a defined amount of time.
package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"
)

type state int

const (
	stateUnknown state = iota
	stateScheduled
	stateCanceled
	stateCompleted
)

// ErrCanceled is returned by [Job.Result] if the scheduled job has been canceled.
var ErrCanceled = errors.New("job canceled")

// Job represents a scheduled job.
type Job struct {
	when   time.Time
	runner Runnable
	err    error
	cancel context.CancelFunc
	done   chan struct{}
	state  state
	lock   sync.RWMutex
}

// Runnable interface for any job to be run by [Schedule].
type Runnable interface {
	Run(context.Context) error
}

// RunFunc is an adaptor type that allows a function to be passed to Schedule as a Runnable.
type RunFunc func(context.Context) error

// Run runs r(ctx)
func (r RunFunc) Run(ctx context.Context) error {
	return r(ctx)
}

// Schedule creates a job for the Runnable, to be executed at the provided time. If ch is not null, Schedule will send a notification to the channel when the job is completed.
func Schedule(ctx context.Context, runner Runnable, delay time.Duration, ch chan struct{}) *Job {
	subCtx, cancel := context.WithCancel(ctx)
	j := Job{
		runner: runner,
		state:  stateScheduled,
		when:   time.Now().Add(delay),
		cancel: cancel,
		done:   ch,
	}
	go j.run(subCtx, delay)
	return &j
}

func (j *Job) run(ctx context.Context, waitTime time.Duration) {
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

// Cancel cancels a scheduled Job.
func (j *Job) Cancel() {
	if j.cancel != nil {
		j.cancel()
	}
}

// Result returns the status of the scheduled job. Returns false if the job has not run yet. If the job has been run,
// the error is the error returned by the Runnable's Run method.  If the job has been canceled, the error will be ErrCanceled.
func (j *Job) Result() (bool, error) {
	s, err, _ := j.getState()
	return s == stateCompleted || s == stateCanceled, err
}

func (j *Job) markCompleted(err error) {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = stateCompleted
	j.err = err
}

func (j *Job) markCanceled() {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.state = stateCanceled
	j.err = ErrCanceled
}

func (j *Job) getState() (state, error, time.Time) {
	j.lock.RLock()
	defer j.lock.RUnlock()
	return j.state, j.err, j.when
}

// Due returns when the job will be run. It returns a zero time if the job has completed.
func (j *Job) Due() time.Time {
	if s, _, when := j.getState(); s == stateScheduled {
		return when
	}
	return time.Time{}
}
