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

func (t MyTask) Run(_ context.Context) error {
	return t.err
}

func TestScheduler_Schedule_Success(t *testing.T) {
	task := &MyTask{}
	job := scheduler.Schedule(context.Background(), task, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := job.Result()
		return done && err == nil
	}, time.Second, 10*time.Millisecond)
}

func TestScheduler_Schedule_Failure(t *testing.T) {
	task := &MyTask{err: fmt.Errorf("failed")}
	job := scheduler.Schedule(context.Background(), task, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := job.Result()
		return done && err != nil
	}, time.Second, 10*time.Millisecond)
}

func TestScheduler_Cancel(t *testing.T) {
	task := &MyTask{}
	job := scheduler.Schedule(context.Background(), task, time.Hour)

	job.Cancel()
	_, err := job.Result()
	require.Error(t, err)
	assert.Equal(t, "job canceled", err.Error())
}
