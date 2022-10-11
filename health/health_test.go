package health

import (
	"context"
	"flag"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/poller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update .golden files")

func TestHandler_Handle(t *testing.T) {
	p := &mocks.Poller{}
	ch := make(chan *poller.Update)
	p.On("Register").Return(ch)
	p.On("Unregister", ch).Return()
	h := Health{Poller: p}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		h.Run(ctx)
		wg.Done()
	}()

	p.On("Refresh").Return().Once()

	resp := httptest.NewRecorder()
	h.Handle(resp, &http.Request{})
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	//p.On("Refresh").Return().Once()
	ch <- &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {
				SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 22.0}},
			},
		},
	}

	assert.Eventually(t, func() bool {
		h.lock.RLock()
		defer h.lock.RUnlock()
		return h.update != nil
	}, time.Second, 10*time.Millisecond)

	resp = httptest.NewRecorder()
	h.Handle(resp, &http.Request{})
	require.Equal(t, http.StatusOK, resp.Code)

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
	wg.Wait()

	mock.AssertExpectationsForObjects(t, p)
}
