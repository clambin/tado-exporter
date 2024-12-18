package health

import (
	"context"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHealth_Handle(t *testing.T) {
	p := mocks.NewPoller(t)
	in, out := makeChannel[poller.Update]()
	p.EXPECT().Subscribe().Return(out)
	p.EXPECT().Unsubscribe(out).Once()
	p.EXPECT().Refresh().Once()

	ctx, cancel := context.WithCancel(context.Background())

	h := New(p, slog.Default())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.Run(ctx)
	}()

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, &http.Request{})
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	in <- poller.Update{}

	assert.Eventually(t, func() bool {
		resp = httptest.NewRecorder()
		h.ServeHTTP(resp, &http.Request{})
		return resp.Code == http.StatusOK
	}, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()
}

func makeChannel[T any]() (chan<- T, <-chan T) {
	ch := make(chan T)
	return ch, ch
}
