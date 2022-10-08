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

type MyJob struct {
	err error
}

func (j MyJob) Run(_ context.Context) error {
	return j.err
}

func TestScheduler_Queue(t *testing.T) {
	s := scheduler.Scheduler{}

	job := &MyJob{}
	s.Schedule(context.Background(), job.Run, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := s.Result()
		return done && err == nil
	}, time.Second, 10*time.Millisecond)

	job = &MyJob{err: fmt.Errorf("failed")}
	s.Schedule(context.Background(), job.Run, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		done, err := s.Result()
		return done && err != nil
	}, time.Second, 10*time.Millisecond)
}

func TestScheduler_Cancel(t *testing.T) {
	s := scheduler.Scheduler{}

	job := &MyJob{}
	s.Schedule(context.Background(), job.Run, time.Hour)

	s.Cancel()

	_, ok := s.Scheduled()
	assert.False(t, ok)
}

func TestScheduler_Scheduled(t *testing.T) {
	s := scheduler.Scheduler{}

	_, ok := s.Scheduled()
	assert.False(t, ok)

	job := &MyJob{}
	s.Schedule(context.Background(), job.Run, time.Hour)

	duration, ok := s.Scheduled()
	require.True(t, ok)
	assert.NotZero(t, duration)

	s.Cancel()
	_, ok = s.Scheduled()
	assert.False(t, ok)

}
