package rules

import (
	"errors"
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/rules/homerules"
	"github.com/clambin/tado-exporter/internal/controller/rules/luart"
	"github.com/clambin/tado-exporter/internal/poller"
	"time"
)

// LoadHomeRules create Rules as per config, for a home.
func LoadHomeRules(config []RuleConfiguration) (Rules, error) {
	r := Rules{
		rules:    make([]Rule, len(config)),
		GetState: GetHomeState,
	}
	for i, cfg := range config {
		var err error
		if r.rules[i], err = LoadHomeRule(cfg); err != nil {
			return Rules{}, fmt.Errorf("failed to load home rule: %w", err)
		}
	}
	return r, nil
}

// GetHomeState returns the State of the home in a poller.Update. Only one home is supported.
func GetHomeState(u poller.Update) (State, error) {
	s := State{
		HomeState: HomeState{
			Overlay: u.HomeState.PresenceLocked != nil && *u.HomeState.PresenceLocked,
			Home:    u.Home(),
		},
		HomeId: *u.HomeBase.Id,
	}
	for dev := range u.GeoTrackedDevices() {
		s.Devices = append(s.Devices, Device{Name: *dev.Name, Home: dev.Location != nil && *dev.Location.AtHome})
	}
	return s, nil
}

type homeRule struct {
	luaScript
	devices set.Set[string]
	args    Args
}

func LoadHomeRule(cfg RuleConfiguration) (Rule, error) {
	s, err := loadLuaScript(cfg.Name, cfg.Script, &homerules.FS)
	if err != nil {
		return nil, &errLua{err: err}
	}
	return homeRule{luaScript: s, devices: set.New(cfg.Users...), args: cfg.Args}, nil
}

func (r homeRule) Evaluate(currentState State) (Action, error) {
	// set up evaluation call
	r.luaScript.State.Global("Evaluate")
	if r.luaScript.State.IsNil(-1) {
		return nil, &errLua{err: errors.New("script does not contain a global Evaluate function")}
	}

	// push arguments
	r.luaScript.pushHomeState(currentState.HomeState)
	r.luaScript.pushDevices(currentState.Devices.filter(r.devices))
	luart.PushMap(r.luaScript.State, r.args)

	// call the script
	if err := r.ProtectedCall(3, 3, 0); err != nil {
		return nil, &errLua{err: err}
	}

	// set up action
	desiredAction := homeAction{HomeId: currentState.HomeId}
	var err error

	// pop the values
	defer r.Pop(3)
	desiredAction.HomeState, err = r.luaScript.getHomeState(-3)
	if err != nil {
		return nil, err
	}
	delay, ok := r.luaScript.State.ToNumber(-2)
	if !ok {
		return nil, &errLuaInvalidResponse{err: errors.New("invalid type: delay")}
	}
	desiredAction.delay = time.Duration(delay) * time.Second
	desiredAction.reason, _ = r.ToString(-1)

	return &desiredAction, nil
}
