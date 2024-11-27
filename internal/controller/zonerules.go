package controller

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/zonerules"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

type zoneState string

const (
	ZoneStateAuto   = zoneState("auto")
	ZoneStateManual = zoneState("manual")
	ZoneStateOff    = zoneState("off")
)

var _ evaluator = zoneRules{}

type zoneRules struct {
	zoneName string
	rules    []evaluator // evaluator, not zoneRule, so we can stub them during unit testing
}

func loadZoneRules(zoneName string, config []RuleConfiguration) (zoneRules, error) {
	// TODO: RuleConfiguration has Users: homeRule needs to include this and only send those users to the script.
	rules := zoneRules{
		zoneName: zoneName,
		rules:    make([]evaluator, 0, len(config)), // evaluator, not zoneRule, so we can stub it out during testing
	}
	for _, cfg := range config {
		r, err := loadLuaScript(cfg.Script, zonerules.FS)
		if err != nil {
			return zoneRules{}, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
		rule, err := newZoneRule(zoneName, r, cfg.Users)
		_ = r.Close()

		if err != nil {
			return zoneRules{}, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
		rules.rules = append(rules.rules, rule)
	}
	return rules, nil
}

func (z zoneRules) ParseUpdate(update poller.Update) (action, error) {
	for _, zone := range update.Zones {
		if *zone.Name == z.zoneName {
			return zoneAction{
				zoneId:    *zone.Zone.Id,
				zoneState: zoneStateFromPollerZone(zone),
			}, nil
		}
	}
	return nil, errors.New("zone not found in update")
}

func (z zoneRules) Evaluate(u update) (action, error) {
	if len(z.rules) == 0 {
		return nil, errors.New("no rules found")
	}

	noChange := make([]zoneAction, 0, len(z.rules))
	change := make([]zoneAction, 0, len(z.rules))
	for i := range z.rules {
		currentState, ok := u.GetZoneState(z.zoneName)
		if !ok {
			return nil, fmt.Errorf("failed to find zone %q in update", z.zoneName)
		}

		a, err := z.rules[i].Evaluate(u)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule %d: %w", i+1, err)
		}
		if a.GetState() == string(currentState) && a.GetDelay() == 0 {
			noChange = append(noChange, a.(zoneAction))
		} else {
			change = append(change, a.(zoneAction))
		}
	}
	if len(change) > 0 {
		slices.SortFunc(change, func(a, b zoneAction) int {
			return cmp.Compare(a.GetDelay(), b.GetDelay())
		})

		return change[0], nil
	}
	return noChange[0], nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ evaluator = zoneRule{}

type zoneRule struct {
	zoneName string
	devices  set.Set[string]
	luaScript
}

func newZoneRule(name string, r io.Reader, devices []string) (zoneRule, error) {
	rule := zoneRule{zoneName: name, devices: set.New(devices...)}
	var err error
	if rule.luaScript, err = newLuaScript(name, r); err != nil {
		return zoneRule{}, err
	}
	return rule, nil
}

func (r zoneRule) Evaluate(u update) (action, error) {
	if err := r.initEvaluation(); err != nil {
		return zoneAction{}, err
	}
	// push arguments
	r.PushString(string(u.GetHomeState()))
	state, ok := u.GetZoneState(r.zoneName)
	if !ok {
		return zoneAction{}, fmt.Errorf("zone %q not found in update", r.zoneName)
	}
	r.PushString(string(state))
	pushDevices(r.luaScript.State, u.GetDevices().filter(r.devices))

	// execute the script
	if err := r.ProtectedCall(3, 3, 0); err != nil {
		return zoneAction{}, fmt.Errorf("lua script failed: %w", err)
	}

	// pop the values
	defer r.Pop(3)
	newState, _ := r.ToString(-3)
	delay, ok := r.ToNumber(-2)
	if !ok {
		return zoneAction{}, fmt.Errorf("invalid type: delay")
	}
	reason, _ := r.ToString(-1)

	return zoneAction{
		zoneState: zoneState(newState),
		delay:     time.Duration(delay) * time.Second,
		reason:    reason,
		homeId:    u.HomeId,
		zoneId:    u.ZoneStates[r.zoneName].ZoneId,
		zoneName:  r.zoneName,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ action = zoneAction{}

type zoneAction struct {
	zoneState zoneState
	TadoClient
	delay    time.Duration
	reason   string
	homeId   tado.HomeId
	zoneId   tado.ZoneId
	zoneName string
}

func (z zoneAction) GetState() string {
	return string(z.zoneState)
}

func (z zoneAction) GetDelay() time.Duration {
	return z.delay
}

func (z zoneAction) GetReason() string {
	return z.reason
}

func (z zoneAction) Do(ctx context.Context, client TadoClient) error {
	switch z.zoneState {
	case ZoneStateAuto:
		resp, err := client.DeleteZoneOverlayWithResponse(ctx, z.homeId, z.zoneId)
		if err == nil && resp.StatusCode() != http.StatusNoContent {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	case ZoneStateOff:
		resp, err := client.SetZoneOverlayWithResponse(ctx, z.homeId, z.zoneId, tado.SetZoneOverlayJSONRequestBody{
			Setting: &tado.ZoneSetting{Type: oapi.VarP(tado.HEATING), Power: oapi.VarP(tado.PowerOFF)},
			Termination: &tado.ZoneOverlayTermination{
				//Type:              oapi.VarP(tado.ZoneOverlayTerminationTypeTIMER),
				TypeSkillBasedApp: oapi.VarP(tado.ZoneOverlayTerminationTypeSkillBasedAppNEXTTIMEBLOCK),
			},
			Type: nil,
		})
		if err == nil && resp.StatusCode() != http.StatusOK {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode())
		}
		return err
	default:
		return fmt.Errorf("invalid zone state: %q", z.zoneState)
	}
}

func (z zoneAction) Description(includeDelay bool) string {
	text := "*" + z.zoneName + "*: "
	if z.zoneState == ZoneStateOff {
		text += "switching off heating"
	} else {
		text += "setting heating to " + string(z.zoneState) + " mode"
	}
	if includeDelay {
		text += " in " + z.delay.String()
	}
	return text
}

func (z zoneAction) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("zone", z.zoneName),
		slog.String("mode", string(z.zoneState)),
		slog.Duration("delay", z.delay),
		slog.String("reason", z.reason),
	)
}
