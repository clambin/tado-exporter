package controller

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"log/slog"
	"net/http"
	"time"
)

type action interface {
	GetState() string
	GetDelay() time.Duration
	GetReason() string
	Description(includeDelay bool) string
	Do(context.Context, TadoClient) error
	slog.LogValuer
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ action = homeAction{}

type homeAction struct {
	state  homeState
	delay  time.Duration
	reason string
	homeId tado.HomeId
}

func (h homeAction) GetState() string {
	return string(h.state)
}

func (h homeAction) GetDelay() time.Duration {
	return h.delay
}

func (h homeAction) GetReason() string {
	return h.reason
}

func (h homeAction) Do(ctx context.Context, client TadoClient) error {
	if h.state == HomeStateAuto {
		resp, err := client.DeletePresenceLockWithResponse(ctx, h.homeId)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
		return nil
	}

	var homePresence tado.HomePresence
	switch h.state {
	case HomeStateHome:
		homePresence = tado.HOME
	case HomeStateAway:
		homePresence = tado.AWAY
	}
	resp, err := client.SetPresenceLockWithResponse(ctx, h.homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &homePresence})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}
	return nil
}

func (h homeAction) Description(includeDelay bool) string {
	text := "Setting home to " + string(h.state) + " mode"
	if includeDelay {
		text += " in " + h.delay.String()
	}
	return text
}

func (h homeAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("action", string(h.state)),
		slog.Duration("delay", h.delay),
		slog.String("reason", h.reason),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ action = zoneAction{}

type zoneAction struct {
	zoneState zoneState
	delay     time.Duration
	reason    string
	homeId    tado.HomeId
	zoneId    tado.ZoneId
	zoneName  string
}

func (z zoneAction) GetState() string {
	return string(z.zoneState)
}

func (z zoneAction) GetDelay() time.Duration {
	return z.delay
}

func (z zoneAction) GetReason() string {
	return z.reason
}

func (z zoneAction) Do(ctx context.Context, client TadoClient) error {
	switch z.zoneState {
	case ZoneStateAuto:
		resp, err := client.DeleteZoneOverlayWithResponse(ctx, z.homeId, z.zoneId)
		if err == nil && resp.StatusCode() != http.StatusNoContent {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	case ZoneStateOff:
		resp, err := client.SetZoneOverlayWithResponse(ctx, z.homeId, z.zoneId, tado.SetZoneOverlayJSONRequestBody{
			Setting: &tado.ZoneSetting{Type: oapi.VarP(tado.HEATING), Power: oapi.VarP(tado.PowerOFF)},
			Termination: &tado.ZoneOverlayTermination{
				//Type:              oapi.VarP(tado.ZoneOverlayTerminationTypeTIMER),
				TypeSkillBasedApp: oapi.VarP(tado.ZoneOverlayTerminationTypeSkillBasedAppNEXTTIMEBLOCK),
			},
			Type: nil,
		})
		if err == nil && resp.StatusCode() != http.StatusOK {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	default:
		return fmt.Errorf("invalid zone state: %q", z.zoneState)
	}
}

func (z zoneAction) Description(includeDelay bool) string {
	text := "*" + z.zoneName + "*: "
	if z.zoneState == ZoneStateOff {
		text += "switching off heating"
	} else {
		text += "setting heating to " + string(z.zoneState) + " mode"
	}
	if includeDelay {
		text += " in " + z.delay.String()
	}
	return text
}

func (z zoneAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("zone", z.zoneName),
		slog.String("mode", string(z.zoneState)),
		slog.Duration("delay", z.delay),
		slog.String("reason", z.reason),
	)
}
