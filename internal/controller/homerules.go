package controller

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/homerules"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func loadHomeRules(config []RuleConfiguration) (rules, error) {
	r := make(rules, len(config))
	for i, cfg := range config {
		var err error
		r[i], err = loadHomeRule(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load home rule: %w", err)
		}
	}
	return r, nil
}

func loadHomeRule(config RuleConfiguration) (rule, error) {
	r, err := loadLuaScript(config.Script, homerules.FS)
	if err != nil {
		return nil, fmt.Errorf("failed to load home rule: %w", err)
	}
	defer func() { _ = r.Close() }()
	return newHomeRule(config.Name, r, config.Users, config.Args)
}

func getHomeStateFromUpdate(u poller.Update) (state, error) {
	s := homeState{
		overlay: u.HomeState.PresenceLocked != nil && *u.HomeState.PresenceLocked,
		home:    u.Home(),
	}
	return s, nil
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ rule = homeRule{}

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

func (r homeRule) Evaluate(u poller.Update) (action, error) {
	if err := r.initEvaluation(); err != nil {
		return nil, err
	}
	// push arguments
	pushHomeState(r.luaScript.State, u)
	pushDevices(r.luaScript.State, u, r.devices)
	luart.PushMap(r.luaScript.State, r.args)

	// execute the script
	if err := r.ProtectedCall(3, 3, 0); err != nil {
		return nil, fmt.Errorf("lua script failed: %w", err)
	}

	// pop the values
	defer r.Pop(3)
	s, err := getHomeState(r.luaScript.State, -3)
	if err != nil {
		return nil, err
	}
	delay, ok := r.ToNumber(-2)
	if !ok {
		return nil, fmt.Errorf("invalid type: delay")
	}
	reason, _ := r.ToString(-1)

	return &homeAction{
		coreAction: coreAction{
			state:  s,
			delay:  time.Duration(delay) * time.Second,
			reason: reason,
		},
		homeId: *u.HomeBase.Id,
	}, nil
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

func (h *homeAction) Do(ctx context.Context, client TadoClient, l *slog.Logger) error {
	if !h.Overlay() {
		l.Debug("removing presenceLock")
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
		l.Debug("setting presenceLock", "lock", string(homePresence))
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

func (h *homeAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("action", h.coreAction.LogValue()),
	)
}

func (h *homeAction) State() state {
	return h.coreAction.state
}
