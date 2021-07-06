package scheduler_test

import (
	"context"
	"github.com/clambin/tado-exporter/controller/scheduler"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	s := scheduler.New()

	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)

	db := &DB{}

	taskA := scheduler.Task{
		ID:   1,
		Run:  RunTask,
		Args: []interface{}{&db.A},
		When: 50 * time.Millisecond,
	}

	taskB := scheduler.Task{
		ID:   2,
		Run:  RunTask,
		Args: []interface{}{&db.B},
		When: 50 * time.Second,
	}

	s.Schedule(ctx, taskA)
	assert.Eventually(t, func() bool { return db.A.Get() == true }, 500*time.Millisecond, 100*time.Millisecond)

	s.Schedule(ctx, taskB)

	assert.Never(t, func() bool { return db.B.Get() == true }, 250*time.Millisecond, 100*time.Millisecond)

	activation, found := s.GetScheduled(2)
	assert.True(t, found)
	assert.NotZero(t, activation)

	s.Cancel(2)
	s.Schedule(ctx, taskB)
	taskB.When = 50 * time.Millisecond
	s.Schedule(ctx, taskB)

	assert.Eventually(t, func() bool { return db.B.Get() == true }, 500*time.Millisecond, 100*time.Millisecond)

	s.Schedule(ctx, taskA)

	taskB.When = 50 * time.Second
	s.Schedule(ctx, taskB)

	cancel()
	assert.Eventually(t, func() bool { return len(s.GetAllScheduled()) == 0 }, 500*time.Millisecond, 50*time.Millisecond)
}

type DB struct {
	A Attrib
	B Attrib
}

type Attrib struct {
	value bool
	lock  sync.RWMutex
}

func (a *Attrib) Get() bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.value
}

func (a *Attrib) Set(value bool) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.value = value
}

func RunTask(_ context.Context, args []interface{}) {
	if len(args) == 0 {
		log.Fatal("missing arguments")
	}
	attrib := args[0].(*Attrib)
	attrib.Set(true)
}
