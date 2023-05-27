package health

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/clambin/go-common/cache"
	"github.com/clambin/tado-exporter/poller"
	"golang.org/x/exp/slog"
	"net/http"
	"time"
)

type Health struct {
	poller.Poller
	cache *cache.Cache[string, *bytes.Buffer]
}

func New(p poller.Poller) *Health {
	return &Health{
		Poller: p,
		cache:  cache.New[string, *bytes.Buffer](15*time.Minute, time.Hour),
	}
}

func (h *Health) Run(ctx context.Context) error {
	slog.Info("health monitor started")
	ch := h.Poller.Register()
	defer h.Poller.Unregister(ch)

	for {
		select {
		case <-ctx.Done():
			slog.Info("health monitor stopped")
			return nil
		case update := <-ch:
			h.store(update)
		}
	}
}

func (h *Health) store(update *poller.Update) {
	b, ok := h.cache.Get("update")
	if !ok {
		b = new(bytes.Buffer)
	} else {
		b.Reset()
	}
	encoder := json.NewEncoder(b)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(update); err != nil {
		panic(err)
	}
	h.cache.Add("update", b)
}

func (h *Health) Handle(w http.ResponseWriter, _ *http.Request) {
	b, ok := h.cache.Get("update")
	if !ok {
		http.Error(w, "no update yet", http.StatusServiceUnavailable)
		h.Poller.Refresh()
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	_, _ = w.Write(b.Bytes())
}
