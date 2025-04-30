package rules

import (
	"context"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado/v2"
	"github.com/clambin/tado/v2/tools"
	"log/slog"
	"net/http"
	"time"
)

type Action interface {
	IsState(state State) bool
	IsAction(action Action) bool
	Delay() time.Duration
	Reason() string
	setReason(reason string)
	Description(includeDelay bool) string
	Do(context.Context, TadoClient, *slog.Logger) error
	slog.LogValuer
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Action = &homeAction{}

type homeAction struct {
	reason    string
	delay     time.Duration
	homeId    tado.HomeId
	homeState HomeState
}

func (h *homeAction) IsState(state State) bool {
	return state.HomeState == h.homeState
}

func (h *homeAction) IsAction(action Action) bool {
	o, ok := action.(*homeAction)
	return ok && o.homeState == h.homeState
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
	text := h.actionString()
	if includeDelay {
		text += " in " + h.delay.String()
	}
	return text
}

func (h *homeAction) actionString() string {
	return "setting home to " + string(homePresences[h.homeState.Home]) + " mode"
}

func (h *homeAction) LogValue() slog.Value {
	return slog.StringValue(h.actionString())
}

func (h *homeAction) Do(ctx context.Context, client TadoClient, l *slog.Logger) error {
	if h.homeState.Overlay {
		return h.setOverlay(ctx, client, l)
	}
	return h.removeOverlay(ctx, client, l)
}

func (h *homeAction) setOverlay(ctx context.Context, client TadoClient, l *slog.Logger) error {
	homePresence := homePresences[h.homeState.Home]
	l.Debug("setting presenceLock", "lock", string(homePresence))
	resp, err := client.SetPresenceLockWithResponse(ctx, h.homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &homePresence})
	if err == nil && resp.StatusCode() != http.StatusNoContent {
		err = tools.HandleErrors(resp.HTTPResponse, map[int]any{
			http.StatusUnauthorized:        resp.JSON401,
			http.StatusForbidden:           resp.JSON403,
			http.StatusUnprocessableEntity: resp.JSON422,
		})
	}
	return err
}

func (h *homeAction) removeOverlay(ctx context.Context, client TadoClient, l *slog.Logger) error {
	l.Debug("removing presenceLock")
	resp, err := client.DeletePresenceLockWithResponse(ctx, h.homeId)
	if err == nil && resp.StatusCode() != http.StatusNoContent {
		err = tools.HandleErrors(resp.HTTPResponse, map[int]any{
			http.StatusUnauthorized:        resp.JSON401,
			http.StatusForbidden:           resp.JSON403,
			http.StatusUnprocessableEntity: resp.JSON422,
		})
	}
	return err
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Action = &zoneAction{}

type zoneAction struct {
	reason    string
	zoneName  string
	delay     time.Duration
	homeId    tado.HomeId
	zoneId    tado.ZoneId
	zoneState ZoneState
}

func (z *zoneAction) IsState(state State) bool {
	return z.zoneState == state.ZoneState
}

func (z *zoneAction) IsAction(action Action) bool {
	o, ok := action.(*zoneAction)
	return ok && o.zoneState == z.zoneState
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
	description := "*" + z.zoneName + "*: " + z.actionString()
	if includeDelay {
		description += " in " + z.delay.String()
	}
	return description
}

var powerMode = map[bool]tado.Power{
	true:  tado.PowerON,
	false: tado.PowerOFF,
}

func (z *zoneAction) actionString() string {
	description := "switching heating "
	switch z.zoneState.Overlay {
	case false:
		description += "to auto mode"
	case true:
		description += zoneStateString[z.zoneState.Heating]
	}
	return description
}

func (z *zoneAction) LogValue() slog.Value {
	return slog.StringValue(z.actionString())
}

func (z *zoneAction) Do(ctx context.Context, client TadoClient, l *slog.Logger) error {
	if z.zoneState.Overlay {
		return z.setOverlay(ctx, client, l)
	}
	return z.removeOverlay(ctx, client, l)
}

func (z *zoneAction) setOverlay(ctx context.Context, client TadoClient, l *slog.Logger) error {
	mode := powerMode[z.zoneState.Heating]
	l.Debug("setting overlay", "mode", string(mode))
	resp, err := client.SetZoneOverlayWithResponse(ctx, z.homeId, z.zoneId, tado.SetZoneOverlayJSONRequestBody{
		Setting: &tado.ZoneSetting{Type: oapi.VarP(tado.HEATING), Power: &mode},
		Termination: &tado.ZoneOverlayTermination{
			Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL),
		},
		Type: nil,
	})
	if err == nil && resp.StatusCode() != http.StatusOK {
		err = tools.HandleErrors(resp.HTTPResponse, map[int]any{
			http.StatusUnauthorized:        resp.JSON401,
			http.StatusForbidden:           resp.JSON403,
			http.StatusUnprocessableEntity: resp.JSON422,
		})
	}
	return err
}

func (z *zoneAction) removeOverlay(ctx context.Context, client TadoClient, l *slog.Logger) error {
	l.Debug("removing overlay")
	resp, err := client.DeleteZoneOverlayWithResponse(ctx, z.homeId, z.zoneId)
	if err == nil && resp.StatusCode() != http.StatusNoContent {
		err = tools.HandleErrors(resp.HTTPResponse, map[int]any{
			http.StatusUnauthorized: resp.JSON401,
			http.StatusForbidden:    resp.JSON403,
		})
	}
	return err
}
