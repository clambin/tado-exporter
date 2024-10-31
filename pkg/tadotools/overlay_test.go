package tadotools

import (
	"context"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSetOverlay(t *testing.T) {
	tests := []struct {
		name        string
		temperature float32
		duration    time.Duration
	}{
		{
			name:        "manual overlay",
			temperature: 19.5,
			duration:    0,
		},
		{
			name:        "timer overlay",
			temperature: 19.5,
			duration:    time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c fakeClient
			ctx := context.Background()
			err := SetOverlay(ctx, &c, 1, 10, tt.temperature, tt.duration)
			assert.NoError(t, err)

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
	body tado.SetZoneOverlayJSONRequestBody
}

func (f *fakeClient) SetZoneOverlayWithResponse(_ context.Context, _ tado.HomeId, _ tado.ZoneId, body tado.SetZoneOverlayJSONRequestBody, _ ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error) {
	f.body = body
	return nil, nil
}
