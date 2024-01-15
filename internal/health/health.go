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
			h.setUpdate(update)
		}
	}
}

func (h *Health) isUpdated() bool {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.updated
}

func (h *Health) setUpdate(update poller.Update) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.update = update
	h.updated = true
}

func (h *Health) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !h.isUpdated() {
		http.Error(w, "no update yet", http.StatusServiceUnavailable)
		h.Poller.Refresh()
		return
	}

	h.lock.RLock()
	defer h.lock.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(h.update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
