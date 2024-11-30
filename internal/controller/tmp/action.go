package tmp

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"log/slog"
	"net/http"
	"time"
)

type action interface {
	GetState() state
	GetDelay() time.Duration
	GetReason() string
	setReason(string)
	Description(includeDelay bool) string
	Do(context.Context, TadoClient) error
	slog.LogValuer
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ action = &homeAction{}

type homeAction struct {
	coreAction
	homeId tado.HomeId
}

var homePresences = map[bool]tado.HomePresence{
	false: tado.AWAY,
	true:  tado.HOME,
}

func (h *homeAction) Do(ctx context.Context, client TadoClient) error {
	home, manual := h.coreAction.state.GetState()
	switch manual {
	case false:
		resp, err := client.DeletePresenceLockWithResponse(ctx, h.homeId)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
	case true:
		homePresence := homePresences[home]
		resp, err := client.SetPresenceLockWithResponse(ctx, h.homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &homePresence})
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ action = &zoneAction{}

type zoneAction struct {
	coreAction
	// TODO: do we need this?
	zoneName string
	homeId   tado.HomeId
	zoneId   tado.ZoneId
}

var zonePower = map[bool]tado.Power{
	true:  tado.PowerON,
	false: tado.PowerOFF,
}

func (z *zoneAction) Do(ctx context.Context, client TadoClient) error {
	on, manual := z.coreAction.state.GetState()
	switch manual {
	case false:
		resp, err := client.DeleteZoneOverlayWithResponse(ctx, z.homeId, z.zoneId)
		if err == nil && resp.StatusCode() != http.StatusNoContent {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	case true:
		resp, err := client.SetZoneOverlayWithResponse(ctx, z.homeId, z.zoneId, tado.SetZoneOverlayJSONRequestBody{
			Setting: &tado.ZoneSetting{Type: oapi.VarP(tado.HEATING), Power: oapi.VarP(zonePower[on])},
			// TODO: think we need this to avoid limitOverlay to immediately switch it off again?
			Termination: &tado.ZoneOverlayTermination{
				TypeSkillBasedApp: oapi.VarP(tado.ZoneOverlayTerminationTypeSkillBasedAppNEXTTIMEBLOCK),
			},
			Type: nil,
		})
		if err == nil && resp.StatusCode() != http.StatusOK {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	}
	return errors.New("should not be reached")
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type coreAction struct {
	state
	reason string
	delay  time.Duration
}

func (a *coreAction) GetState() state {
	return a.state
}

func (a *coreAction) GetDelay() time.Duration {
	return a.delay
}

func (a *coreAction) GetReason() string {
	return a.reason
}

func (a *coreAction) setReason(s string) {
	a.reason = s
}

func (a *coreAction) Description(includeDelay bool) string {
	text := a.state.Description()
	if includeDelay {
		text += " in " + a.delay.String()
	}
	return text
}

func (a *coreAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("state", a.state),
		slog.Duration("delay", a.delay),
		slog.String("reason", a.reason),
	)
}
