package rules

import (
	"codeberg.org/clambin/go-common/set"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/zonerules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"time"
)

// LoadZoneRules create Rules as per config, for a zone named zoneName.
func LoadZoneRules(zoneName string, config []RuleConfiguration) (Rules, error) {
	r := Rules{
		rules:    make([]Rule, len(config)),
		GetState: GetZoneState(zoneName),
	}
	for i, cfg := range config {
		var err error
		if r.rules[i], err = LoadZoneRule(zoneName, cfg); err != nil {
			return Rules{}, fmt.Errorf("failed to load home rule: %w", err)
		}
	}
	return r, nil
}

// GetZoneState returns a function that returns the State of a zone, named zoneName, from a poller.Update.
func GetZoneState(zoneName string) func(u poller.Update) (State, error) {
	return func(u poller.Update) (State, error) {
		zone, ok := u.Zones.GetZone(zoneName)
		if !ok {
			return State{}, fmt.Errorf("zone %q not found in update", zoneName)
		}
		s := State{
			HomeState: HomeState{
				Overlay: u.HomeState.PresenceLocked != nil && *u.HomeState.PresenceLocked,
				Home:    u.Home(),
			},
			ZoneState: ZoneState{
				Overlay: zone.ZoneState.Overlay != nil && zone.ZoneState.Overlay.Termination != nil && *zone.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeMANUAL,
				Heating: zone.ZoneState.Setting != nil && *zone.ZoneState.Setting.Power == tado.PowerON,
			},
			HomeId: *u.HomeBase.Id,
			ZoneId: *zone.Zone.Id,
		}
		for dev := range u.MobileDevices.GeoTrackedDevices() {
			s.Devices = append(s.Devices, Device{Name: *dev.Name, Home: dev.Location != nil && *dev.Location.AtHome})
		}
		return s, nil
	}
}

var _ Rule = zoneRule{}

type zoneRule struct {
	luaScript
	devices  set.Set[string]
	args     Args
	zoneName string
}

func LoadZoneRule(zoneName string, cfg RuleConfiguration) (Rule, error) {
	s, err := loadLuaScript(cfg.Name, cfg.Script, &zonerules.FS)
	if err != nil {
		return nil, &errLua{err: err}
	}
	return zoneRule{zoneName: zoneName, luaScript: s, devices: set.New(cfg.Users...), args: cfg.Args}, nil
}

func (r zoneRule) Evaluate(currentState State) (Action, error) {
	// set up evaluation call
	r.Global("Evaluate")
	if r.IsNil(-1) {
		return nil, &errLua{err: errors.New("script does not contain an Evaluate function")}
	}

	// push arguments
	r.pushHomeState(currentState.HomeState)
	r.pushZoneState(currentState.ZoneState)
	r.pushDevices(currentState.Devices.filter(r.devices))
	r.pushArgs(r.args)

	// execute the script
	if err := r.ProtectedCall(4, 3, 0); err != nil {
		return nil, &errLua{err: err}
	}

	// set up action
	desiredAction := zoneAction{homeId: currentState.HomeId, zoneId: currentState.ZoneId, zoneName: r.zoneName}
	var err error

	// pop the values
	defer r.Pop(3)
	desiredAction.zoneState, err = r.getZoneState(-3)
	if err != nil {
		return nil, err
	}
	delay, ok := r.ToNumber(-2)
	if !ok {
		return nil, &errLuaInvalidResponse{err: errors.New("invalid delay value")}
	}
	desiredAction.delay = time.Duration(delay) * time.Second
	desiredAction.reason, _ = r.ToString(-1)

	return &desiredAction, nil
}
