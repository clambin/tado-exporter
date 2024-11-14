package bot

import (
	"context"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	mockPoller "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"log/slog"
	"testing"
	"time"
)

func TestBot_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := mocks.NewSocketModeHandler(t)
	h.EXPECT().HandleSlashCommand(mock.Anything, mock.Anything)
	h.EXPECT().HandleDefault(mock.Anything).Once()
	h.EXPECT().RunEventLoopContext(ctx).Return(nil).Once()
	p := mockPoller.NewPoller(t)
	ch := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(ch).Once()
	p.EXPECT().Unsubscribe(ch).Once()

	b := New(nil, h, p, nil, slog.Default())

	errCh := make(chan error)
	go func() { errCh <- b.Run(ctx) }()

	_, ok := b.getUpdate()
	assert.False(t, ok)

	ch <- poller.Update{}

	assert.Eventually(t, func() bool {
		_, ok = b.getUpdate()
		return ok
	}, time.Second, time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}
