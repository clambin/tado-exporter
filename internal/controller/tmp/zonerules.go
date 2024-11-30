package tmp

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/controller/zonerules"
	"io"
	"time"
)

func loadZoneRules(config []RuleConfiguration) ([]evaluator, error) {
	rules := make([]evaluator, len(config))
	for i, cfg := range config {
		var err error
		if rules[i], err = loadZoneRule(cfg); err != nil {
			return nil, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
	}
	return rules, nil
}

func loadZoneRule(config RuleConfiguration) (evaluator, error) {
	r, err := loadLuaScript(config.Script, zonerules.FS)
	if err != nil {
		return nil, fmt.Errorf("failed to load home rule: %w", err)
	}
	defer func() { _ = r.Close() }()
	return newZoneRule(config.Name, r, config.Users, config.Args)
}

func getZoneStateFromUpdate(name string) func(update) (state, error) {
	return func(u update) (state, error) {
		return u.ZoneStates[name].zoneState, nil
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ evaluator = zoneRule{}

type zoneRule struct {
	luaScript
	devices  set.Set[string]
	args     Args
	zoneName string
}

func newZoneRule(name string, r io.Reader, devices []string, args Args) (*zoneRule, error) {
	script, err := newLuaScript(name, r)
	if err != nil {
		return nil, err
	}
	return &zoneRule{luaScript: script, zoneName: name, devices: set.New(devices...), args: args}, nil
}

func (r zoneRule) Evaluate(u update) (action, error) {
	// set up Evaluate call
	if err := r.initEvaluation(); err != nil {
		return nil, err
	}

	// push arguments
	u.GetHomeState().ToLua(r.luaScript.State)
	s, ok := u.GetZoneState(r.zoneName)
	if !ok {
		return nil, fmt.Errorf("zone %q not found in update", r.zoneName)
	}
	s.ToLua(r.luaScript.State)
	u.GetDevices().filter(r.devices).toLua(r.luaScript.State)
	luart.PushMap(r.State, r.args)

	// execute the script
	if err := r.ProtectedCall(4, 3, 0); err != nil {
		return nil, fmt.Errorf("lua script failed: %w", err)
	}

	// pop the values
	defer r.Pop(3)
	newState, err := toZoneState(r.luaScript.State, -3)
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
			state:  newState,
			delay:  time.Duration(delay) * time.Second,
			reason: reason,
		},
		homeId:   u.HomeId,
		zoneId:   u.ZoneStates[r.zoneName].ZoneId,
		zoneName: r.zoneName,
	}, nil
}
