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
	State() state
	Delay() time.Duration
	Reason() string
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
	if !h.Overlay() {
		resp, err := client.DeletePresenceLockWithResponse(ctx, h.homeId)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
		return nil
	} else {
		homePresence := homePresences[h.Mode()]
		resp, err := client.SetPresenceLockWithResponse(ctx, h.homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &homePresence})
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
		return nil
	}
}

func (h *homeAction) Description(includeDelay bool) string {
	return "setting home to " + h.coreAction.Description(includeDelay)
}

func (h *homeAction) State() state {
	return h.coreAction.state
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ action = &zoneAction{}

type zoneAction struct {
	coreAction
	zoneName string
	homeId   tado.HomeId
	zoneId   tado.ZoneId
}

var powerMode = map[bool]tado.Power{
	true:  tado.PowerON,
	false: tado.PowerOFF,
}

func (z *zoneAction) Do(ctx context.Context, client TadoClient) error {
	if !z.State().Overlay() {
		resp, err := client.DeleteZoneOverlayWithResponse(ctx, z.homeId, z.zoneId)
		if err == nil && resp.StatusCode() != http.StatusNoContent {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	} else {
		mode := powerMode[z.State().Mode()]
		resp, err := client.SetZoneOverlayWithResponse(ctx, z.homeId, z.zoneId, tado.SetZoneOverlayJSONRequestBody{
			Setting: &tado.ZoneSetting{Type: oapi.VarP(tado.HEATING), Power: &mode},
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
	}
}

func (z *zoneAction) Description(includeDelay bool) string {
	return "*" + z.zoneName + "*: switching heating " + z.coreAction.Description(includeDelay)
}

func (z *zoneAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("zone", z.zoneName),
		slog.Any("action", z.coreAction.LogValue()),
	)
}
func (h *zoneAction) State() state {
	return h.coreAction.state
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type coreAction struct {
	state
	reason string
	delay  time.Duration
}

func (a *coreAction) Delay() time.Duration {
	return a.delay
}

func (a *coreAction) Reason() string {
	return a.reason
}
func (a *coreAction) setReason(reason string) {
	a.reason = reason
}
func (a *coreAction) Description(includeDelay bool) string {
	text := a.state.String()
	if includeDelay {
		text += " in " + a.delay.String()
	}
	return text
}

func (a *coreAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("state", a.state.LogValue()),
		slog.Duration("delay", a.delay),
		slog.String("reason", a.reason),
	)
}
