package processor

import (
	"context"
	"errors"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/testutil"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	a := action.Action{
		Delay:  100 * time.Millisecond,
		Reason: "foo",
		Label:  "bar",
		State:  testutil.FakeState{ModeValue: action.ZoneInAutoMode},
	}

	ctx := context.Background()
	ch := make(chan struct{})
	task := newTask(ctx, nil, a, ch)

	<-ch
	completed, err := task.job.Result()
	assert.NoError(t, err)
	assert.True(t, completed)
}

func TestTask_Stress(t *testing.T) {
	const actionCount = 1e3

	var wg sync.WaitGroup
	wg.Add(actionCount)
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < actionCount; i++ {
		go func() {
			defer wg.Done()
			a := action.Action{Delay: time.Hour}
			task := newTask(ctx, nil, a, nil)

			assert.Eventually(t, func() bool {
				completed, err := task.job.Result()
				return completed && errors.Is(err, scheduler.ErrCanceled)
			}, time.Second, 100*time.Millisecond)

		}()
	}
	cancel()
	wg.Wait()
}

func TestTask_scheduledBefore(t *testing.T) {
	ctx := context.Background()
	a := action.Action{Delay: time.Hour}
	task := newTask(ctx, nil, a, nil)
	go task.job.Run(a.Delay)
	defer task.job.Cancel()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for i := 1; i < 1600; i++ {
		<-ticker.C
		a2 := action.Action{Delay: time.Until(task.job.Due())}
		assert.True(t, task.scheduledBefore(a2))
	}
}
