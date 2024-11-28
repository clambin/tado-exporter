package controller

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/controller/zonerules"
	"io"
	"time"
)

type zoneState string

const (
	ZoneStateAuto   = zoneState("auto")
	ZoneStateManual = zoneState("manual")
	ZoneStateOff    = zoneState("off")
)

func loadZoneRules(zoneName string, config []RuleConfiguration) ([]evaluator, error) {
	rules := make([]evaluator, len(config))
	for i, cfg := range config {
		r, err := loadLuaScript(cfg.Script, zonerules.FS)
		if err != nil {
			return nil, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
		rules[i], err = newZoneRule(zoneName, r, cfg.Users, cfg.Args)
		_ = r.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
	}
	return rules, nil
}

func getZoneStateFromUpdate(zoneName string) func(update) (action, error) {
	return func(u update) (action, error) {
		z, ok := u.ZoneStates[zoneName]
		if !ok {
			return nil, fmt.Errorf("zone %q not found in update", zoneName)
		}
		return &zoneAction{zoneId: z.ZoneId, zoneState: z.zoneState}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ evaluator = zoneRule{}

type zoneRule struct {
	zoneName string
	devices  set.Set[string]
	args     Args
	luaScript
}

func newZoneRule(name string, r io.Reader, devices []string, args Args) (zoneRule, error) {
	rule := zoneRule{zoneName: name, devices: set.New(devices...), args: args}
	var err error
	if rule.luaScript, err = newLuaScript(name, r); err != nil {
		return zoneRule{}, err
	}
	return rule, nil
}

func (r zoneRule) Evaluate(u update) (action, error) {
	// set up Evaluate call
	if err := r.initEvaluation(); err != nil {
		return nil, err
	}

	// push arguments
	r.PushString(string(u.GetHomeState()))
	state, ok := u.GetZoneState(r.zoneName)
	if !ok {
		return &zoneAction{}, fmt.Errorf("zone %q not found in update", r.zoneName)
	}
	r.PushString(string(state))
	pushDevices(r.luaScript.State, u.GetDevices().filter(r.devices))
	luart.PushMap(r.State, r.args)

	// execute the script
	if err := r.ProtectedCall(4, 3, 0); err != nil {
		return nil, fmt.Errorf("lua script failed: %w", err)
	}

	// pop the values
	defer r.Pop(3)
	newState, _ := r.ToString(-3)
	delay, ok := r.ToNumber(-2)
	if !ok {
		return nil, fmt.Errorf("invalid type: delay")
	}
	reason, _ := r.ToString(-1)

	return &zoneAction{
		zoneState: zoneState(newState),
		delay:     time.Duration(delay) * time.Second,
		reason:    reason,
		homeId:    u.HomeId,
		zoneId:    u.ZoneStates[r.zoneName].ZoneId,
		zoneName:  r.zoneName,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
