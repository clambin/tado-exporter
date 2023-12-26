package health

import (
	"context"
	"encoding/json"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
	"net/http"
	"sync"
)

type Health struct {
	poller.Poller
	logger  *slog.Logger
	update  poller.Update
	updated bool
	lock    sync.RWMutex
}

func New(p poller.Poller, logger *slog.Logger) *Health {
	return &Health{
		Poller: p,
		logger: logger,
	}
}

func (h *Health) Run(ctx context.Context) error {
	h.logger.Debug("started")
	defer h.logger.Debug("stopped")

	ch := h.Poller.Subscribe()
	defer h.Poller.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			h.lock.Lock()
			h.update = update
			h.updated = true
			h.lock.Unlock()
		}
	}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if !h.updated {
		http.Error(w, "no update yet", http.StatusServiceUnavailable)
		h.Poller.Refresh()
		return
	}

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(h.update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
