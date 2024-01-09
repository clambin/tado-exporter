package processor

import (
	"context"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTask_scheduledBefore(t *testing.T) {
	ctx := context.Background()
	a := action.Action{Delay: time.Hour}
	task := newTask(ctx, nil, a, nil)
	defer task.job.Cancel()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for i := 1; i < 1600; i++ {
		<-ticker.C
		a2 := action.Action{Delay: time.Until(task.job.Due())}
		assert.True(t, task.scheduledBefore(a2))
	}
}
