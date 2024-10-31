package tadotools

import (
	"context"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"time"
)

type TadoClient interface {
	SetZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, body tado.SetZoneOverlayJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error)
}

func SetOverlay(ctx context.Context, c TadoClient, homeId tado.HomeId, zoneId tado.ZoneId, temperature float32, duration time.Duration) error {
	// possibly set power to "off" if temp <= 5?
	req := tado.SetZoneOverlayJSONRequestBody{
		Setting: &tado.ZoneSetting{
			Power:       oapi.VarP(tado.PowerON),
			Type:        oapi.VarP(tado.HEATING),
			Temperature: &tado.Temperature{Celsius: oapi.VarP(temperature)},
		},
		Termination: &tado.ZoneOverlayTermination{
			Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL),
		},
	}
	if duration > 0 {
		req.Termination.Type = oapi.VarP(tado.ZoneOverlayTerminationTypeTIMER)
		req.Termination.DurationInSeconds = oapi.VarP(int(duration.Seconds()))
	}
	_, err := c.SetZoneOverlayWithResponse(ctx, homeId, zoneId, req)
	return err
}
