package scheduler

import (
	"context"
	"sync"
	"time"
)

type Scheduler struct {
	task *Task
}

func (s *Scheduler) Schedule(ctx context.Context, job func(ctx context.Context) error, waitTime time.Duration) {
	s.Cancel()

	ctx2, cancel := context.WithCancel(ctx)
	s.task = &Task{
		state:  stateUnknown,
		job:    job,
		when:   time.Now().Add(waitTime),
		cancel: cancel,
	}
	go s.task.Run(ctx2, waitTime)
}

func (s *Scheduler) Cancel() {
	if s.task != nil {
		s.task.cancel()
		s.task = nil
	}
}

func (s *Scheduler) Result() (completed bool, err error) {
	if s.task == nil {
		return
	}
	var result state
	result, err = s.task.getState()
	if completed = result.done(); completed {
		s.Cancel()
	}
	return
}

func (s *Scheduler) Scheduled() (duration time.Duration, ok bool) {
	if s.task == nil {
		return
	}
	return time.Until(s.task.when), true
}

type state int

const (
	stateUnknown state = iota
	stateScheduled
	stateCompleted
	stateFailed
)

func (s state) done() bool {
	return s == stateCompleted || s == stateFailed
}

type Task struct {
	job    func(ctx context.Context) error
	when   time.Time
	state  state
	err    error
	cancel context.CancelFunc
	lock   sync.RWMutex
}

func (t *Task) Run(ctx context.Context, waitTime time.Duration) {
	t.setState(stateScheduled, nil)
	select {
	case <-ctx.Done():
		return
	case <-time.After(waitTime):
		err := t.job(ctx)
		s := stateCompleted
		if err != nil {
			s = stateFailed
		}
		t.setState(s, err)
	}
}

func (t *Task) setState(state state, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.state = state
	t.err = err
}

func (t *Task) getState() (state state, err error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.state, t.err
}
