package health

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/clambin/go-common/cache"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
	"net/http"
	"time"
)

type Health struct {
	poller.Poller
	logger *slog.Logger
	cache  *cache.Cache[string, *bytes.Buffer]
}

func New(p poller.Poller, logger *slog.Logger) *Health {
	return &Health{
		Poller: p,
		logger: logger,
		cache:  cache.New[string, *bytes.Buffer](15*time.Minute, time.Hour),
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
			h.store(update)
		}
	}
}

// TODO:  just store update and marshal it when we actually get called. way more efficient!
func (h *Health) store(update poller.Update) {
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
