package health

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/clambin/tado-exporter/internal/poller"
)

type Health struct {
	poller   Poller
	updated  atomic.Value
	logger   *slog.Logger
	interval time.Duration
}

type Poller interface {
	Subscribe() <-chan poller.Update
	Unsubscribe(<-chan poller.Update)
	Refresh()
}

func New(p Poller, interval time.Duration, logger *slog.Logger) *Health {
	return &Health{
		poller:   p,
		interval: interval,
		logger:   logger,
	}
}

func (h *Health) Run(ctx context.Context) {
	h.logger.Debug("started")
	defer h.logger.Debug("stopped")

	ch := h.poller.Subscribe()
	defer h.poller.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			h.updated.Store(time.Now())
		}
	}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	const maxMissedUpdates = 5
	lastUpdate := h.updated.Load()
	if lastUpdate == nil || time.Since(lastUpdate.(time.Time)) > maxMissedUpdates*h.interval {
		http.Error(w, "no update yet", http.StatusServiceUnavailable)
		h.poller.Refresh()
	}
}
