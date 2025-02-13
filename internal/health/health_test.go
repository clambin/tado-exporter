package health

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealth_Handle(t *testing.T) {
	var subscribed atomic.Bool

	ch := make(chan poller.Update)
	p := mocks.NewPoller(t)
	p.EXPECT().Subscribe().RunAndReturn(func() <-chan poller.Update {
		subscribed.Store(true)
		return ch
	}).Once()
	p.EXPECT().Unsubscribe((<-chan poller.Update)(ch)).Run(func(_ <-chan poller.Update) {
		subscribed.Store(false)
	}).Maybe()
	p.EXPECT().Refresh().Once()

	h := New(p, slog.New(slog.DiscardHandler))
	go func() {
		h.Run(t.Context())
		assert.False(t, subscribed.Load())
	}()

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, &http.Request{})
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	ch <- poller.Update{}

	assert.Eventually(t, func() bool {
		resp = httptest.NewRecorder()
		h.ServeHTTP(resp, &http.Request{})
		return resp.Code == http.StatusOK
	}, time.Second, 10*time.Millisecond)
}
