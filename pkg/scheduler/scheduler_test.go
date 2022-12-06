package scheduler_test

import (
	"context"
	"errors"
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

func TestSchedule_Success(t *testing.T) {
	var task MyTask
	job := scheduler.Schedule(context.Background(), &task, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := job.Result()
		return done && err == nil
	}, time.Second, 10*time.Millisecond)
}

func TestSchedule_Failure(t *testing.T) {
	task := MyTask{err: fmt.Errorf("failed")}
	job := scheduler.Schedule(context.Background(), &task, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		completed, err := job.Result()
		return completed && err != nil
	}, time.Second, 10*time.Millisecond)

	_, err := job.Result()
	require.Error(t, err)
	assert.Equal(t, "failed", err.Error())
}

func TestJob_Cancel(t *testing.T) {
	var task MyTask
	job := scheduler.Schedule(context.Background(), &task, time.Hour)

	job.Cancel()

	assert.Eventually(t, func() bool {
		completed, err := job.Result()
		return completed && errors.Is(err, scheduler.ErrCanceled)
	}, time.Second, 10*time.Millisecond)
}

func TestJob_Cancel_Chained(t *testing.T) {
	var task MyTask
	ctx, cancel := context.WithCancel(context.Background())
	job := scheduler.Schedule(ctx, &task, time.Hour)

	cancel()

	assert.Eventually(t, func() bool {
		completed, err := job.Result()
		return completed && errors.Is(err, scheduler.ErrCanceled)
	}, time.Second, 10*time.Millisecond)
}
