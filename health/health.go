package health

import (
	"context"
	"encoding/json"
	"github.com/clambin/tado-exporter/poller"
	"net/http"
	"sync"
	"time"
)

type Handler struct {
	poller.Poller
	update *poller.Update
	ch     chan *poller.Update
	lock   sync.RWMutex
}

func (h *Handler) Run(ctx context.Context) {
	h.ch = make(chan *poller.Update)
	h.Register(h.ch)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case update := <-h.ch:
			h.lock.Lock()
			h.update = update
			h.lock.Unlock()
		}
	}

	h.Unregister(h.ch)
}

func (h *Handler) Handle(w http.ResponseWriter, _ *http.Request) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if h.update == nil {
		http.Error(w, "no update yet", http.StatusServiceUnavailable)
		h.Poller.Refresh()
		return
	}

	lastUpdate := h.GetLastUpdate()
	if time.Since(lastUpdate) > time.Hour {
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
