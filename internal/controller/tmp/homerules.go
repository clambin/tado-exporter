package tmp

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/homerules"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"io"
	"time"
)

func loadHomeRules(config []RuleConfiguration) ([]evaluator, error) {
	rules := make([]evaluator, len(config))
	for i, cfg := range config {
		var err error
		rules[i], err = loadHomeRule(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load home rule: %w", err)
		}
	}
	return rules, nil
}

func loadHomeRule(config RuleConfiguration) (evaluator, error) {
	r, err := loadLuaScript(config.Script, homerules.FS)
	if err != nil {
		return nil, fmt.Errorf("failed to load home rule: %w", err)
	}
	defer func() { _ = r.Close() }()
	return newHomeRule(config.Name, r, config.Users, config.Args)
}

func getHomeStateFromUpdate(u update) (state, error) {
	return u.homeState, nil
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ evaluator = homeRule{}

type homeRule struct {
	luaScript
	devices set.Set[string]
	args    Args
}

func newHomeRule(name string, r io.Reader, devices []string, args Args) (*homeRule, error) {
	script, err := newLuaScript(name, r)
	if err != nil {
		return nil, err
	}
	return &homeRule{luaScript: script, devices: set.New[string](devices...), args: args}, nil
}

func (r homeRule) Evaluate(u update) (action, error) {
	if err := r.initEvaluation(); err != nil {
		return nil, err
	}
	// push arguments
	u.GetHomeState().ToLua(r.luaScript.State)
	u.GetDevices().filter(r.devices).toLua(r.luaScript.State)
	luart.PushMap(r.luaScript.State, r.args)

	// execute the script
	if err := r.ProtectedCall(3, 3, 0); err != nil {
		return nil, fmt.Errorf("lua script failed: %w", err)
	}

	// pop the values
	defer r.Pop(3)
	s, err := toHomeState(r.luaScript.State, -3)
	if err != nil {
		return nil, err
	}
	delay, ok := r.ToNumber(-2)
	if !ok {
		return nil, fmt.Errorf("invalid type: delay")
	}
	reason, _ := r.ToString(-1)

	return &homeAction{
		coreAction: coreAction{state: s,
			delay:  time.Duration(delay) * time.Second,
			reason: reason,
		},
		homeId: u.HomeId,
	}, nil
}
