package scheduler_test

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type MyTask struct {
	err error
}

var _ scheduler.Runnable = &MyTask{}

func (t MyTask) Run(_ context.Context) error {
	return t.err
}

func TestSchedule(t *testing.T) {
	ch := make(chan struct{})
	var task MyTask
	job := scheduler.Schedule(context.Background(), &task, 100*time.Millisecond, ch)

	<-ch
	done, err := job.Result()
	require.NoError(t, err)
	assert.True(t, done)
}

func TestSchedule_Stress(t *testing.T) {
	const jobCount = 1e5
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for range int(jobCount) {
		wg.Add(1)
		job := scheduler.Schedule(ctx, &MyTask{}, time.Hour, nil)
		go func() {
			time.Sleep(100 * time.Millisecond)
			job.Cancel()
			wg.Done()
		}()
	}
	cancel()
	wg.Wait()
}

func TestSchedule_Failure(t *testing.T) {
	ch := make(chan struct{})
	task := MyTask{err: fmt.Errorf("failed")}
	job := scheduler.Schedule(context.Background(), &task, 100*time.Millisecond, ch)

	<-ch
	_, err := job.Result()
	require.Error(t, err)
	assert.Equal(t, "failed", err.Error())
}

func TestJob_Cancel(t *testing.T) {
	ch := make(chan struct{})
	var task MyTask
	job := scheduler.Schedule(context.Background(), &task, time.Hour, ch)

	job.Cancel()
	<-ch
	completed, err := job.Result()
	assert.True(t, completed)
	assert.ErrorIs(t, err, scheduler.ErrCanceled)
}

func TestJob_Cancel_Chained(t *testing.T) {
	ch := make(chan struct{})
	var task MyTask
	ctx, cancel := context.WithCancel(context.Background())
	job := scheduler.Schedule(ctx, &task, time.Hour, ch)

	cancel()
	<-ch
	completed, err := job.Result()
	assert.True(t, completed)
	assert.ErrorIs(t, err, scheduler.ErrCanceled)
}

func TestJob_TimeToFire(t *testing.T) {
	var task MyTask
	ctx, cancel := context.WithCancel(context.Background())
	job := scheduler.Schedule(ctx, &task, time.Hour, nil)

	assert.Eventually(t, func() bool {
		state, _, _ := job.GetState()
		return state == scheduler.StateScheduled
	}, time.Second, time.Millisecond)

	assert.Equal(t, 60*time.Minute, time.Until(job.Due()).Round(time.Minute))

	cancel()
}
