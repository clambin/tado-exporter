package tadotools

import (
	"context"
	"fmt"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestSetOverlay(t *testing.T) {
	tests := []struct {
		name        string
		temperature float32
		duration    time.Duration
		statusCode  int
		eval        []evalFunc
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name:        "manual overlay (heating)",
			temperature: 19.5,
			eval:        []evalFunc{evalHeating(), evalHeatingOn(19.5), evalManualOverlay()},
			wantErr:     assert.NoError,
		},
		{
			name:    "manual overlay (off)",
			eval:    []evalFunc{evalHeating(), evalHeatingOff(), evalManualOverlay()},
			wantErr: assert.NoError,
		},
		{
			name:        "timer overlay (heating)",
			temperature: 19.5,
			duration:    time.Hour,
			eval:        []evalFunc{evalHeating(), evalHeatingOn(19.5), evalTimerOverlay(time.Hour)},
			wantErr:     assert.NoError,
		},
		{
			name:     "timer overlay (off)",
			duration: time.Hour,
			eval:     []evalFunc{evalHeating(), evalHeatingOff(), evalTimerOverlay(time.Hour)},
			wantErr:  assert.NoError,
		},
		{
			name:        "failure",
			temperature: 19.5,
			duration:    time.Hour,
			statusCode:  http.StatusBadRequest,
			wantErr:     assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			c := fakeClient{statusCode: tt.statusCode, evaluate: tt.eval}
			err := SetOverlay(ctx, &c, 1, 10, tt.temperature, tt.duration)
			tt.wantErr(t, err)
		})
	}
}

type evalFunc func(tado.SetZoneOverlayJSONRequestBody) error

func evalHeating() evalFunc {
	return func(req tado.SetZoneOverlayJSONRequestBody) error {
		if *req.Setting.Type != tado.HEATING {
			return fmt.Errorf("invalid type: %s", *req.Setting.Type)
		}
		return nil
	}
}

func evalHeatingOn(temperature float32) evalFunc {
	return func(req tado.SetZoneOverlayJSONRequestBody) error {
		if *req.Setting.Power != tado.PowerON || req.Setting.Temperature == nil || *req.Setting.Temperature.Celsius != temperature {
			return fmt.Errorf("invalid heating parameters")
		}
		return nil
	}
}

func evalHeatingOff() evalFunc {
	return func(req tado.SetZoneOverlayJSONRequestBody) error {
		if *req.Setting.Power != tado.PowerOFF {
			return fmt.Errorf("invalid heating parameters")
		}
		return nil
	}
}

func evalManualOverlay() evalFunc {
	return func(req tado.SetZoneOverlayJSONRequestBody) error {
		if req.Termination == nil || *req.Termination.Type != tado.ZoneOverlayTerminationTypeMANUAL {
			return fmt.Errorf("invalid termination type")
		}
		return nil
	}
}

func evalTimerOverlay(duration time.Duration) evalFunc {
	return func(req tado.SetZoneOverlayJSONRequestBody) error {
		if req.Termination == nil || *req.Termination.Type != tado.ZoneOverlayTerminationTypeTIMER {
			return fmt.Errorf("invalid termination type")
		}
		if *req.Termination.DurationInSeconds != int(duration.Seconds()) {
			return fmt.Errorf("invalid termination time: %v", time.Duration(*req.Termination.DurationInSeconds)*time.Second)
		}
		return nil
	}
}

type fakeClient struct {
	evaluate   []evalFunc
	statusCode int
	status     string
}

func (f *fakeClient) SetZoneOverlayWithResponse(_ context.Context, _ tado.HomeId, _ tado.ZoneId, req tado.SetZoneOverlayJSONRequestBody, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
	for _, eval := range f.evaluate {
		if err := eval(req); err != nil {
			return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: http.StatusUnprocessableEntity, Status: err.Error()}}, nil
		}
	}
	if f.statusCode == 0 {
		f.statusCode = http.StatusOK
		f.status = http.StatusText(f.statusCode)
	}
	return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: f.statusCode, Status: f.status}}, nil
}
