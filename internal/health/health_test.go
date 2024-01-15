package health

import (
	"context"
	"flag"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update .golden files")

func TestHealth_Handle(t *testing.T) {
	p := mocks.NewPoller(t)
	ch := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(ch)
	p.EXPECT().Unsubscribe(ch)
	p.EXPECT().Refresh().Once()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)

	h := New(p, slog.Default())
	go func() { errCh <- h.Run(ctx) }()

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, &http.Request{})
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	ch <- poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {
				SensorDataPoints: tado.ZoneInfoSensorDataPoints{InsideTemperature: tado.Temperature{Celsius: 22.0}},
			},
		},
	}

	assert.Eventually(t, func() bool {
		return h.isUpdated()
	}, time.Second, 10*time.Millisecond)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, &http.Request{})
	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))

	response := resp.Body.Bytes()

	gp := filepath.Join("testdata", t.Name()+".golden")
	if *update {
		err := os.WriteFile(gp, response, 0644)
		require.NoError(t, err)
	}

	golden, err := os.ReadFile(gp)
	require.NoError(t, err)
	assert.Equal(t, string(golden), string(response))

	cancel()
	assert.NoError(t, <-errCh)
}

func BenchmarkHealth_Handle(b *testing.B) {
	p := mocks.Poller{}
	p.EXPECT().Refresh()

	ch := make(chan poller.Update)
	p.EXPECT().Subscribe().Return(ch)
	p.EXPECT().Unsubscribe(ch)

	h := New(&p, slog.Default())
	h.setUpdate(poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {
				SensorDataPoints: tado.ZoneInfoSensorDataPoints{InsideTemperature: tado.Temperature{Celsius: 22.0}},
			},
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := httptest.NewRecorder()
		h.ServeHTTP(resp, &http.Request{})
	}
}
