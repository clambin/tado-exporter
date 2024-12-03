package controller

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/controller/zonerules"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func loadZoneRules(zoneName string, config []RuleConfiguration) (rules, error) {
	r := make(rules, len(config))
	for i, cfg := range config {
		var err error
		r[i], err = loadZoneRule(zoneName, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
	}
	return r, nil
}

func loadZoneRule(zoneName string, config RuleConfiguration) (rule, error) {
	r, err := loadLuaScript(config.Script, zonerules.FS)
	if err != nil {
		return nil, fmt.Errorf("failed to load zone rule: %w", err)
	}
	defer func() { _ = r.Close() }()
	return newZoneRule(zoneName, r, config.Users, config.Args)

}

func getZoneStateFromUpdate(zoneName string) func(poller.Update) (state, error) {
	return func(u poller.Update) (state, error) {
		zone, ok := u.GetZone(zoneName)
		if !ok {
			return nil, fmt.Errorf("zone %q not found in update", zoneName)
		}
		s := zoneState{
			overlay: zone.ZoneState.Overlay != nil && zone.ZoneState.Overlay.Termination != nil && *zone.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeMANUAL,
			heating: zone.ZoneState.Setting != nil && *zone.ZoneState.Setting.Power == tado.PowerON,
		}
		return s, nil
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ rule = zoneRule{}

type zoneRule struct {
	luaScript
	zoneName string
	devices  set.Set[string]
	args     Args
}

func newZoneRule(name string, r io.Reader, devices []string, args Args) (*zoneRule, error) {
	rule := zoneRule{zoneName: name, devices: set.New(devices...), args: args}
	var err error
	if rule.luaScript, err = newLuaScript(name, r); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r zoneRule) Evaluate(u poller.Update) (action, error) {
	// set up Evaluate call
	if err := r.initEvaluation(); err != nil {
		return nil, err
	}

	zone, ok := u.GetZone(r.zoneName)
	if !ok {
		return nil, fmt.Errorf("zone %q not found in update", r.zoneName)
	}

	// push arguments
	pushHomeState(r.luaScript.State, u)
	pushZoneState(r.luaScript.State, zone)
	pushDevices(r.luaScript.State, u, r.devices)
	luart.PushMap(r.State, r.args)

	// execute the script
	if err := r.ProtectedCall(4, 3, 0); err != nil {
		return nil, fmt.Errorf("lua script failed: %w", err)
	}

	// pop the values
	defer r.Pop(3)
	s, err := getZoneState(r.luaScript.State, -3)
	if err != nil {
		return nil, err
	}
	delay, ok := r.ToNumber(-2)
	if !ok {
		return nil, fmt.Errorf("invalid type: delay")
	}
	reason, _ := r.ToString(-1)

	return &zoneAction{
		coreAction: coreAction{
			state:  s,
			delay:  time.Duration(delay) * time.Second,
			reason: reason,
		},
		homeId:   *u.HomeBase.Id,
		zoneId:   *zone.Zone.Id,
		zoneName: r.zoneName,
	}, nil
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

func (z *zoneAction) Do(ctx context.Context, client TadoClient, l *slog.Logger) error {
	if !z.State().Overlay() {
		l.Debug("removing overlay")
		resp, err := client.DeleteZoneOverlayWithResponse(ctx, z.homeId, z.zoneId)
		if err == nil && resp.StatusCode() != http.StatusNoContent {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	} else {
		mode := powerMode[z.State().Mode()]
		l.Debug("setting overlay", "mode", string(mode))
		resp, err := client.SetZoneOverlayWithResponse(ctx, z.homeId, z.zoneId, tado.SetZoneOverlayJSONRequestBody{
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
