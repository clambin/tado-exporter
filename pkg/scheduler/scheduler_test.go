package scheduler_test

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type MyTask struct {
	err error
}

func (t MyTask) Run(_ context.Context) error {
	return t.err
}

func TestScheduler_Queue(t *testing.T) {
	task := &MyTask{}
	job := scheduler.Schedule(context.Background(), task, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := job.Result()
		return done && err == nil
	}, time.Second, 10*time.Millisecond)

	task = &MyTask{err: fmt.Errorf("failed")}
	job = scheduler.Schedule(context.Background(), task, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := job.Result()
		return done && err != nil
	}, time.Second, 10*time.Millisecond)
}

func TestScheduler_Cancel(t *testing.T) {
	task := &MyTask{}
	job := scheduler.Schedule(context.Background(), task, time.Hour)

	job.Cancel()
	completed, err := job.Result()
	assert.NoError(t, err)
	assert.True(t, completed)
}
