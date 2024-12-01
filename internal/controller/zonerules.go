package controller

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/controller/zonerules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"io"
	"time"
)

func loadZoneRules(zoneName string, config []RuleConfiguration) ([]evaluator, error) {
	rules := make([]evaluator, len(config))
	for i, cfg := range config {
		var err error
		rules[i], err = loadZoneRule(zoneName, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
	}
	return rules, nil
}

func loadZoneRule(zoneName string, config RuleConfiguration) (evaluator, error) {
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

var _ evaluator = zoneRule{}

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
