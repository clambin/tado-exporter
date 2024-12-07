package scheduler_test

import (
	"context"
	"errors"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestSchedule(t *testing.T) {
	ch := make(chan struct{})

	f := scheduler.RunFunc(func(ctx context.Context) error { return nil })
	job := scheduler.Schedule(context.Background(), f, 100*time.Millisecond, ch)

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
		f := scheduler.RunFunc(func(ctx context.Context) error { return nil })
		job := scheduler.Schedule(ctx, f, time.Hour, nil)
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
	f := scheduler.RunFunc(func(_ context.Context) error { return errors.New("failed") })
	job := scheduler.Schedule(context.Background(), f, 100*time.Millisecond, ch)

	<-ch
	_, err := job.Result()
	require.Error(t, err)
	assert.Equal(t, "failed", err.Error())
}

func TestJob_Cancel(t *testing.T) {
	ch := make(chan struct{})
	f := scheduler.RunFunc(func(_ context.Context) error { return nil })
	job := scheduler.Schedule(context.Background(), f, time.Hour, ch)

	job.Cancel()
	<-ch
	completed, err := job.Result()
	assert.True(t, completed)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestJob_Cancel_Chained(t *testing.T) {
	ch := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	f := scheduler.RunFunc(func(_ context.Context) error { return nil })
	job := scheduler.Schedule(ctx, f, time.Hour, ch)

	cancel()
	<-ch
	completed, err := job.Result()
	assert.True(t, completed)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestJob_Due(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := scheduler.RunFunc(func(_ context.Context) error { return nil })
	job := scheduler.Schedule(ctx, f, time.Hour, nil)
	assert.Equal(t, 60*time.Minute, job.Due().Round(time.Minute))

	cancel()
}
