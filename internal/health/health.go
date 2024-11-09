package health

import (
	"context"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
	"net/http"
	"sync/atomic"
)

type Health struct {
	poller.Poller
	logger  *slog.Logger
	updated atomic.Bool
}

func New(p poller.Poller, logger *slog.Logger) *Health {
	return &Health{
		Poller: p,
		logger: logger,
	}
}

func (h *Health) Run(ctx context.Context) {
	h.logger.Debug("started")
	defer h.logger.Debug("stopped")

	ch := h.Poller.Subscribe()
	defer h.Poller.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			h.updated.Store(true)
		}
	}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !h.updated.Load() {
		http.Error(w, "no update yet", http.StatusServiceUnavailable)
		h.Poller.Refresh()
		return
	}
}
