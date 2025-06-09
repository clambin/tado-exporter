package health

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/stretchr/testify/assert"
)

func TestHealth_Handle(t *testing.T) {
	p := fakePoller{ch: make(chan poller.Update)}
	h := New(&p, 30*time.Second, slog.New(slog.DiscardHandler))
	go func() {
		h.Run(t.Context())
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

var _ Poller = &fakePoller{}

type fakePoller struct {
	ch chan poller.Update
}

func (f *fakePoller) Subscribe() <-chan poller.Update {
	return f.ch
}

func (f *fakePoller) Unsubscribe(_ <-chan poller.Update) {
}

func (f *fakePoller) Refresh() {
	f.ch <- poller.Update{}
}
