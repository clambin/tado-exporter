package rules

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"log/slog"
	"net/http"
	"time"
)

type Action interface {
	IsState(state State) bool
	IsActionState(action Action) bool
	Delay() time.Duration
	Reason() string
	setReason(reason string)
	Description(includeDelay bool) string
	Do(context.Context, TadoClient, *slog.Logger) error
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Action = &homeAction{}

type homeAction struct {
	reason string
	delay  time.Duration
	tado.HomeId
	HomeState
}

func (h *homeAction) IsState(state State) bool {
	return state.HomeState == h.HomeState
}

func (h *homeAction) IsActionState(action Action) bool {
	o, ok := action.(*homeAction)
	return ok && o.HomeState == h.HomeState
}

func (h *homeAction) Delay() time.Duration {
	return h.delay
}

func (h *homeAction) Reason() string {
	return h.reason
}

func (h *homeAction) setReason(reason string) {
	h.reason = reason
}

var homePresences = map[bool]tado.HomePresence{
	false: tado.AWAY,
	true:  tado.HOME,
}

func (h *homeAction) Description(includeDelay bool) string {
	text := "setting home to " + string(homePresences[h.HomeState.Home]) + " mode"
	if includeDelay {
		text += " in " + h.delay.String()
	}
	return text
}

func (h *homeAction) Do(ctx context.Context, client TadoClient, l *slog.Logger) error {
	if !h.HomeState.Overlay {
		l.Debug("removing presenceLock")
		resp, err := client.DeletePresenceLockWithResponse(ctx, h.HomeId)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
		return nil
	} else {
		homePresence := homePresences[h.HomeState.Home]
		l.Debug("setting presenceLock", "lock", string(homePresence))
		resp, err := client.SetPresenceLockWithResponse(ctx, h.HomeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &homePresence})
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Action = &zoneAction{}

type zoneAction struct {
	reason   string
	zoneName string
	delay    time.Duration
	tado.HomeId
	tado.ZoneId
	ZoneState
}

func (z *zoneAction) IsState(state State) bool {
	return z.ZoneState == state.ZoneState
}

func (z *zoneAction) IsActionState(action Action) bool {
	o, ok := action.(*zoneAction)
	return ok && o.ZoneState == z.ZoneState
}

func (z *zoneAction) Delay() time.Duration {
	return z.delay
}

func (z *zoneAction) Reason() string {
	return z.reason
}

func (z *zoneAction) setReason(reason string) {
	z.reason = reason
}

var zoneStateString = map[bool]string{
	true:  "on",
	false: "off",
}

func (z *zoneAction) Description(includeDelay bool) string {
	description := "*" + z.zoneName + "*: switching heating "
	switch z.ZoneState.Overlay {
	case false:
		description += "to auto mode"
	case true:
		description += zoneStateString[z.ZoneState.Heating]

	}
	if includeDelay {
		description += " in " + z.delay.String()
	}
	return description
}

var powerMode = map[bool]tado.Power{
	true:  tado.PowerON,
	false: tado.PowerOFF,
}

func (z *zoneAction) Do(ctx context.Context, client TadoClient, l *slog.Logger) error {
	if !z.ZoneState.Overlay {
		l.Debug("removing overlay")
		resp, err := client.DeleteZoneOverlayWithResponse(ctx, z.HomeId, z.ZoneId)
		if err == nil && resp.StatusCode() != http.StatusNoContent {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	} else {
		mode := powerMode[z.ZoneState.Heating]
		l.Debug("setting overlay", "mode", string(mode))
		resp, err := client.SetZoneOverlayWithResponse(ctx, z.HomeId, z.ZoneId, tado.SetZoneOverlayJSONRequestBody{
			Setting: &tado.ZoneSetting{Type: oapi.VarP(tado.HEATING), Power: &mode},
			Termination: &tado.ZoneOverlayTermination{
				Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL),
			},
			Type: nil,
		})
		if err == nil && resp.StatusCode() != http.StatusOK {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	}
}
