package controller

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/homerules"
	"io"
	"time"
)

type homeState string

const (
	HomeStateAuto = homeState("auto")
	HomeStateHome = homeState("home")
	HomeStateAway = homeState("away")
)

func loadHomeRules(config []RuleConfiguration) ([]evaluator, error) {
	rules := make([]evaluator, len(config))
	for i, cfg := range config {
		r, err := loadLuaScript(cfg.Script, homerules.FS)
		if err != nil {
			return nil, fmt.Errorf("failed to load home rule: %w", err)
		}
		rules[i], err = newHomeRule(cfg.Name, r, cfg.Users)
		_ = r.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to load home rule: %w", err)
		}
	}
	return rules, nil
}

func getHomeStateFromUpdate(u update) (action, error) {
	return &homeAction{homeId: u.HomeId, state: u.homeState}, nil
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ evaluator = homeRule{}

type homeRule struct {
	luaScript
	devices set.Set[string]
}

func newHomeRule(name string, r io.Reader, devices []string) (*homeRule, error) {
	rule := homeRule{devices: set.New[string](devices...)}
	var err error
	if rule.luaScript, err = newLuaScript(name, r); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r homeRule) Evaluate(u update) (action, error) {
	if err := r.initEvaluation(); err != nil {
		return nil, err
	}
	// push arguments
	r.PushString(string(u.GetHomeState()))
	pushDevices(r.luaScript.State, u.GetDevices().filter(r.devices))

	// execute the script
	if err := r.ProtectedCall(2, 3, 0); err != nil {
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

	return &homeAction{
		state:  homeState(newState),
		delay:  time.Duration(delay) * time.Second,
		reason: reason,
		homeId: u.HomeId,
	}, nil
}
