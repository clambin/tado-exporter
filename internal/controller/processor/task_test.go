package processor

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	const actionCount = 1e3

	var wg sync.WaitGroup
	wg.Add(actionCount)
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < actionCount; i++ {
		go func() {
			defer wg.Done()
			a := action.Action{Delay: time.Hour}
			task := newTask(ctx, nil, a, nil)

			for {
				completed, _ := task.job.Result()
				if completed {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}

				time.Sleep(time.Second)
			}
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
