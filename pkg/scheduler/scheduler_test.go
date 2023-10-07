package scheduler_test

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type MyTask struct {
	err error
}

var _ scheduler.Task = &MyTask{}

func (t MyTask) Run(_ context.Context) error {
	return t.err
}

func TestSchedule(t *testing.T) {
	var task MyTask
	job := scheduler.Schedule(context.Background(), &task, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := job.Result()
		return done && err == nil
	}, time.Second, 10*time.Millisecond)
}

func TestScheduleWithNotification(t *testing.T) {
	ch := make(chan struct{})
	var task MyTask
	job := scheduler.ScheduleWithNotification(context.Background(), &task, 100*time.Millisecond, ch)

	<-ch
	done, err := job.Result()
	require.NoError(t, err)
	assert.True(t, done)
}

func TestSchedule_Failure(t *testing.T) {
	ch := make(chan struct{})
	task := MyTask{err: fmt.Errorf("failed")}
	job := scheduler.ScheduleWithNotification(context.Background(), &task, 100*time.Millisecond, ch)

	<-ch
	_, err := job.Result()
	require.Error(t, err)
	assert.Equal(t, "failed", err.Error())
}

func TestJob_Cancel(t *testing.T) {
	ch := make(chan struct{})
	var task MyTask
	job := scheduler.ScheduleWithNotification(context.Background(), &task, time.Hour, ch)

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
	job := scheduler.ScheduleWithNotification(ctx, &task, time.Hour, ch)

	cancel()
	<-ch
	completed, err := job.Result()
	assert.True(t, completed)
	assert.ErrorIs(t, err, scheduler.ErrCanceled)
}

func TestJob_TimeToFire(t *testing.T) {
	ch := make(chan struct{})
	var task MyTask
	ctx, cancel := context.WithCancel(context.Background())
	job := scheduler.ScheduleWithNotification(ctx, &task, time.Hour, ch)

	assert.Eventually(t, func() bool {
		state, _, _ := job.GetState()
		return state == scheduler.StateScheduled
	}, time.Second, time.Millisecond)

	assert.Equal(t, 60*time.Minute, job.TimeToFire().Round(time.Minute))

	cancel()
}
