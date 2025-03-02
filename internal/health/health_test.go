package health

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealth_Handle(t *testing.T) {
	p := fakePoller{ch: make(chan poller.Update)}
	h := New(&p, 30*time.Second, slog.New(slog.DiscardHandler))
	go func() {
		h.Run(t.Context())
		assert.Zero(t, p.subscribed.Load())
	}()

	// no update yet: not up
	assert.False(t, isUp(h))

	// receive update: eventually we're marked as up
	go p.Refresh()
	assert.Eventually(t, func() bool { return isUp(h) }, time.Second, time.Millisecond)

	// no update after 5*interval: not up
	h.updated.Store(time.Time{})
	assert.False(t, isUp(h))
}

func isUp(h http.Handler) bool {
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, &http.Request{})
	return resp.Code == http.StatusOK
}

var _ poller.Poller = &fakePoller{}

type fakePoller struct {
	ch         chan poller.Update
	subscribed atomic.Int32
	refreshed  atomic.Int32
}

func (f *fakePoller) Subscribe() <-chan poller.Update {
	f.subscribed.Add(1)
	return f.ch
}

func (f *fakePoller) Unsubscribe(_ <-chan poller.Update) {
	f.subscribed.Add(-1)
}

func (f *fakePoller) Refresh() {
	f.ch <- poller.Update{}
}
