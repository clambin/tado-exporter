package controller

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/homerules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
)

type HomeState string

const (
	HomeStateAuto = HomeState("auto")
	HomeStateHome = HomeState("home")
	HomeStateAway = HomeState("away")
)

var _ Evaluator = HomeRules{}

type HomeRules []HomeRule

func (h HomeRules) ParseUpdate(update poller.Update) (Action, error) {
	return homeAction{
		state:  HomeState(strings.ToLower(string(*update.Presence))),
		homeId: *update.HomeBase.Id,
	}, nil
}

func LoadHomeRules(config []RuleConfiguration) (HomeRules, error) {
	var rules HomeRules
	for _, cfg := range config {
		r, err := loadLuaScript(cfg.Script, homerules.FS)
		if err != nil {
			return nil, fmt.Errorf("failed to load home rule: %w", err)
		}
		rule, err := NewHomeRule(cfg.Name, r)
		_ = r.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to load home rule: %w", err)
		}
		rules = append(rules, *rule)
	}
	return rules, nil
}

func (h HomeRules) Evaluate(u Update) (Action, error) {
	if len(h) == 0 {
		return nil, errors.New("no rules found")
	}

	noChange := make([]homeAction, 0, len(h))
	change := make([]homeAction, 0, len(h))
	for i := range h {
		currentState := u.GetHomeState()

		a, err := h[i].Evaluate(u)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule %d: %w", i+1, err)
		}
		if a.GetState() == string(currentState) && a.GetDelay() == 0 {
			noChange = append(noChange, a.(homeAction))
		} else {
			change = append(change, a.(homeAction))
		}
	}
	if len(change) > 0 {
		slices.SortFunc(change, func(a, b homeAction) int {
			return cmp.Compare(a.GetDelay(), b.GetDelay())
		})

		return change[0], nil
	}
	return noChange[0], nil
}

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
var _ Evaluator = HomeRule{}

type HomeRule struct {
	luaScript
}

func NewHomeRule(name string, r io.Reader) (*HomeRule, error) {
	var rule HomeRule
	var err error
	if rule.luaScript, err = newLuaScript(name, r); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r HomeRule) Evaluate(u Update) (Action, error) {
	if err := r.initEvaluation(); err != nil {
		return nil, err
	}
	// push arguments
	r.PushString(string(u.GetHomeState()))
	pushDevices(r.luaScript.State, u.GetDevices())

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

	return homeAction{
		state:  HomeState(newState),
		delay:  time.Duration(delay) * time.Second,
		reason: reason,
		homeId: u.HomeId,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Action = homeAction{}

type homeAction struct {
	state  HomeState
	delay  time.Duration
	reason string
	homeId tado.HomeId
}

func (h homeAction) GetState() string {
	return string(h.state)
}

func (h homeAction) GetDelay() time.Duration {
	return h.delay
}

func (h homeAction) GetReason() string {
	return h.reason
}

func (h homeAction) Do(ctx context.Context, client TadoClient) error {
	if h.state == HomeStateAuto {
		resp, err := client.DeletePresenceLockWithResponse(ctx, h.homeId)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		}
		return nil
	}

	var homePresence tado.HomePresence
	switch h.state {
	case HomeStateHome:
		homePresence = tado.HOME
	case HomeStateAway:
		homePresence = tado.AWAY
	}
	resp, err := client.SetPresenceLockWithResponse(ctx, h.homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: &homePresence})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}
	return nil
}

func (h homeAction) Description(includeDelay bool) string {
	text := "Setting home to " + string(h.state) + " mode"
	if includeDelay {
		text += " in " + h.delay.String()
	}
	return text
}

func (h homeAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("action", string(h.state)),
		slog.Duration("delay", h.delay),
		slog.String("reason", h.reason),
	)
}
