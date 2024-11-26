package controller

import (
	"cmp"
	"context"
	"errors"
	"fmt"
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

type ZoneState string

const (
	ZoneStateAuto   = ZoneState("auto")
	ZoneStateManual = ZoneState("manual")
	ZoneStateOff    = ZoneState("off")
)

var _ Evaluator = ZoneRules{}

type ZoneRules struct {
	zoneName string
	rules    []Evaluator // Evaluator, not ZoneRule, so we can stub them during unit testing
}

func LoadZoneRules(zoneName string, config []RuleConfiguration) (ZoneRules, error) {
	zoneRules := ZoneRules{
		zoneName: zoneName,
		rules:    make([]Evaluator, 0, len(config)), // Evaluator, not ZoneRule, so we can stub it out during testing
	}
	for _, cfg := range config {
		r, err := loadLuaScript(cfg.Script, zonerules.FS)
		if err != nil {
			return ZoneRules{}, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
		rule, err := NewZoneRule(zoneName, r)
		_ = r.Close()

		if err != nil {
			return ZoneRules{}, fmt.Errorf("failed to load zone rule %q: %w", cfg.Name, err)
		}
		zoneRules.rules = append(zoneRules.rules, rule)
	}
	return zoneRules, nil
}

func (z ZoneRules) ParseUpdate(update poller.Update) (Action, error) {
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

func (z ZoneRules) Evaluate(u Update) (Action, error) {
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

var _ Evaluator = ZoneRule{}

type ZoneRule struct {
	zoneName string
	luaScript
}

func NewZoneRule(name string, r io.Reader) (ZoneRule, error) {
	rule := ZoneRule{zoneName: name}
	var err error
	if rule.luaScript, err = newLuaScript(name, r); err != nil {
		return ZoneRule{}, err
	}
	return rule, nil
}

func (r ZoneRule) Evaluate(u Update) (Action, error) {
	if err := r.initEvaluation(); err != nil {
		return zoneAction{}, err
	}
	// push arguments
	r.PushString(string(u.GetHomeState()))
	zoneState, ok := u.GetZoneState(r.zoneName)
	if !ok {
		return zoneAction{}, fmt.Errorf("zone %q not found in update", r.zoneName)
	}
	r.PushString(string(zoneState))
	pushDevices(r.luaScript.State, u.GetDevices())

	// execute the script
	if err := r.ProtectedCall(3, 3, 0); err != nil {
		return zoneAction{}, fmt.Errorf("lua script failed: %w", err)
	}

	// pop the values
	defer r.Pop(3)
	newZoneState, _ := r.ToString(-3)
	delay, ok := r.ToNumber(-2)
	if !ok {
		return zoneAction{}, fmt.Errorf("invalid type: delay")
	}
	reason, _ := r.ToString(-1)

	return zoneAction{
		zoneState: ZoneState(newZoneState),
		delay:     time.Duration(delay) * time.Second,
		reason:    reason,
		homeId:    u.HomeId,
		zoneId:    u.ZoneStates[r.zoneName].ZoneId,
		zoneName:  r.zoneName,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ Action = zoneAction{}

type zoneAction struct {
	zoneState ZoneState
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
			Setting:     &tado.ZoneSetting{Power: oapi.VarP(tado.PowerOFF)},
			Termination: &tado.ZoneOverlayTermination{Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL)},
			Type:        nil,
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
	text := z.zoneName + ": setting heating to " + string(z.zoneState) + " mode"
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
