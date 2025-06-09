package bot

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBot_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := mocks.NewSocketModeHandler(t)
	h.EXPECT().HandleSlashCommand(mock.Anything, mock.Anything)
	h.EXPECT().HandleInteraction(mock.Anything, mock.Anything)
	h.EXPECT().HandleDefault(mock.Anything).Once()
	h.EXPECT().RunEventLoopContext(ctx).Return(nil).Once()
	p := mocks.NewPoller(t)
	in, out := makeChannel[poller.Update]()
	p.EXPECT().Subscribe().Return(out).Once()
	p.EXPECT().Unsubscribe(out).Once()

	b := New(nil, h, p, nil, slog.Default())

	errCh := make(chan error)
	go func() { errCh <- b.Run(ctx) }()

	_, ok := b.commandRunner.getUpdate()
	assert.False(t, ok)

	in <- poller.Update{}

	assert.Eventually(t, func() bool {
		_, ok = b.commandRunner.getUpdate()
		return ok
	}, time.Second, time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}

func makeChannel[T any]() (chan<- T, <-chan T) {
	ch := make(chan T)
	return ch, ch
}
