package health

import (
	"context"
	"encoding/json"
	"github.com/clambin/tado-exporter/poller"
	"net/http"
	"sync"
	"time"
)

type Health struct {
	poller.Poller
	update     *poller.Update
	lastUpdate time.Time
	lock       sync.RWMutex
}

func (h *Health) Run(ctx context.Context) {
	ch := h.Register()
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case update := <-ch:
			h.lock.Lock()
			h.update = update
			h.lastUpdate = time.Now()
			h.lock.Unlock()
		}
	}
	h.Unregister(ch)
}

func (h *Health) Handle(w http.ResponseWriter, _ *http.Request) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if time.Since(h.lastUpdate) > time.Hour {
		http.Error(w, "no update yet", http.StatusServiceUnavailable)
		h.Poller.Refresh()
		return
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(h.update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
