package health

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/poller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHandler_Handle(t *testing.T) {
	p := &mocks.Poller{}
	p.On("Register", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	p.On("Unregister", mock.AnythingOfType("chan *poller.Update")).Return(nil)
	h := Handler{Poller: p, Ch: make(chan *poller.Update)}

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

	h.Ch <- &poller.Update{
		Zones: map[int]tado.Zone{1: {ID: 1, Name: "foo"}},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {
				SensorDataPoints: tado.ZoneInfoSensorDataPoints{Temperature: tado.Temperature{Celsius: 22.0}},
			},
		},
	}
	p.On("GetLastUpdate").Return(time.Now().Add(-24 * time.Hour)).Once()
	p.On("Refresh").Return().Once()

	resp = httptest.NewRecorder()
	h.Handle(resp, &http.Request{})
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	p.On("GetLastUpdate").Return(time.Now()).Once()

	resp = httptest.NewRecorder()
	h.Handle(resp, &http.Request{})
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, `{
  "Zones": {
    "1": {
      "id": 1,
      "name": "foo",
      "devices": null
    }
  },
  "ZoneInfo": {
    "1": {
      "setting": {
        "power": "",
        "temperature": {
          "celsius": 0
        }
      },
      "activityDataPoints": {
        "heatingPower": {
          "percentage": 0
        }
      },
      "sensorDataPoints": {
        "insideTemperature": {
          "celsius": 22
        },
        "humidity": {
          "percentage": 0
        }
      },
      "openwindow": {
        "detectedTime": "0001-01-01T00:00:00Z",
        "durationInSeconds": 0,
        "expiry": "0001-01-01T00:00:00Z",
        "remainingTimeInSeconds": 0
      },
      "overlay": {
        "type": "",
        "setting": {
          "type": "",
          "power": "",
          "temperature": {
            "celsius": 0
          }
        },
        "termination": {
          "type": ""
        }
      }
    }
  },
  "UserInfo": null,
  "WeatherInfo": {
    "outsideTemperature": {
      "celsius": 0
    },
    "solarIntensity": {
      "percentage": 0
    },
    "weatherState": {
      "value": ""
    }
  }
}
`, resp.Body.String())

	cancel()
	wg.Wait()

	mock.AssertExpectationsForObjects(t, p)
}
