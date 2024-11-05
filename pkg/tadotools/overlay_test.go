package tadotools

import (
	"context"
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
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name:        "manual overlay",
			temperature: 19.5,
			duration:    0,
			wantErr:     assert.NoError,
		},
		{
			name:        "timer overlay",
			temperature: 19.5,
			duration:    time.Hour,
			wantErr:     assert.NoError,
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
			c := fakeClient{statusCode: tt.statusCode}
			err := SetOverlay(ctx, &c, 1, 10, tt.temperature, tt.duration)
			tt.wantErr(t, err)

			if err != nil {
				return
			}

			assert.Equal(t, tado.PowerON, *c.body.Setting.Power)
			assert.Equal(t, tado.HEATING, *c.body.Setting.Type)
			assert.Equal(t, tt.temperature, *c.body.Setting.Temperature.Celsius)
			if tt.duration > 0 {
				assert.Equal(t, tado.ZoneOverlayTerminationTypeTIMER, *c.body.Termination.Type)
				assert.Equal(t, int(tt.duration.Seconds()), *c.body.Termination.DurationInSeconds)
			} else {
				assert.Equal(t, tado.ZoneOverlayTerminationTypeMANUAL, *c.body.Termination.Type)
			}
		})
	}
}

var _ TadoClient = &fakeClient{}

type fakeClient struct {
	body       tado.SetZoneOverlayJSONRequestBody
	statusCode int
	status     string
}

func (f *fakeClient) SetZoneOverlayWithResponse(_ context.Context, _ tado.HomeId, _ tado.ZoneId, body tado.SetZoneOverlayJSONRequestBody, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
	f.body = body
	if f.statusCode == 0 {
		f.statusCode = http.StatusOK
		f.status = "200 OK"
	}
	return &tado.SetZoneOverlayResponse{HTTPResponse: &http.Response{StatusCode: f.statusCode, Status: f.status}}, nil
}
