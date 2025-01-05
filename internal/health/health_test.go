package health

import (
	"context"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealth_Handle(t *testing.T) {
	p := mocks.NewPoller(t)
	ch := make(chan poller.Update)
	var in chan<- poller.Update = ch
	var out <-chan poller.Update = ch
	p.EXPECT().Subscribe().Return(out)
	p.EXPECT().Unsubscribe(out).Once()
	p.EXPECT().Refresh().Once()

	ctx, cancel := context.WithCancel(context.Background())

	h := New(p, slog.Default())
	done := make(chan struct{})
	go func() {
		h.Run(ctx)
		done <- struct{}{}
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
	<-done
}
