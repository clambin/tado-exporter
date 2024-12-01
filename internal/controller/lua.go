package controller

import (
	"embed"
	"errors"
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"io"
	"os"
	"strings"
)

// loadLuaScript opens a Lua script from disk, an embedded file system, or as text.
func loadLuaScript(cfg ScriptConfig, fs embed.FS) (io.ReadCloser, error) {
	switch {
	case cfg.Text != "":
		return io.NopCloser(strings.NewReader(cfg.Text)), nil
	case cfg.Packaged != "":
		return fs.Open(cfg.Packaged)
	case cfg.Path != "":
		return os.Open(cfg.Path)
	default:
		return nil, fmt.Errorf("script config is empty")
	}
}

type luaScript struct {
	*lua.State
}

func newLuaScript(name string, r io.Reader) (luaScript, error) {
	var script luaScript
	var err error
	script.State, err = luart.Compile(name, r)
	return script, err
}

func (r luaScript) initEvaluation() error {
	const evalName = "Evaluate"
	r.Global(evalName)
	if r.IsNil(-1) {
		return fmt.Errorf("lua does not contain %s function", evalName)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func pushHomeState(l *lua.State, u poller.Update) {
	luaObject := map[string]any{
		"Overlay": u.HomeState.PresenceLocked != nil && *u.HomeState.PresenceLocked,
		"Home":    u.Home(),
	}
	luart.PushMap(l, luaObject)
}

func getHomeState(l *lua.State, index int) (homeState, error) {
	if !l.IsTable(index) {
		return homeState{}, &errLuaInvalidResponse{fmt.Errorf("no table found at index %d", index)}
	}
	// more idiomatic for TableToMap to push these on the stack and then pop them with ToBoolean?
	obj := luart.TableToMap(l, l.AbsIndex(index))
	var s homeState
	var err error
	if s.overlay, err = getAttribute[bool](obj, "Overlay"); err != nil {
		return homeState{}, &errLuaInvalidResponse{fmt.Errorf("homeState.Overlay: %w", err)}
	}
	if s.home, err = getAttribute[bool](obj, "Home"); err != nil {
		return homeState{}, &errLuaInvalidResponse{err: fmt.Errorf("homeState.Home: %w", err)}
	}
	return s, nil
}

func pushZoneState(l *lua.State, zone poller.Zone) {
	luaObject := map[string]any{
		"Overlay": zone.ZoneState.Overlay != nil && zone.ZoneState.Overlay.Termination != nil && *zone.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeMANUAL,
		"Heating": zone.ZoneState.Setting.Power != nil && *zone.ZoneState.Setting.Power == tado.PowerON,
	}
	luart.PushMap(l, luaObject)
}

func getZoneState(l *lua.State, index int) (zoneState, error) {
	if !l.IsTable(index) {
		return zoneState{}, &errLuaInvalidResponse{fmt.Errorf("no table found at index %d", index)}
	}
	// more idiomatic for TableToMap to push these on the stack and then pop them with ToBoolean?
	obj := luart.TableToMap(l, l.AbsIndex(index))
	var s zoneState
	var err error
	if s.overlay, err = getAttribute[bool](obj, "Overlay"); err != nil {
		return zoneState{}, &errLuaInvalidResponse{fmt.Errorf("zoneState.Overlay: %w", err)}
	}
	if s.heating, err = getAttribute[bool](obj, "Heating"); err != nil {
		return zoneState{}, &errLuaInvalidResponse{fmt.Errorf("zoneState.Heating: %w", err)}
	}
	return s, nil
}

func pushDevices(l *lua.State, u poller.Update, users set.Set[string]) {
	l.NewTable()
	var count int
	for d := range u.GeoTrackedDevices() {
		if len(users) > 0 && !users.Contains(*d.Name) {
			continue
		}
		count++
		l.NewTable()
		l.PushString(*d.Name) // Push the value for "Name"
		l.SetField(-2, "Name")
		l.PushBoolean(*d.Location.AtHome) // Push the value for "AtHome"
		l.SetField(-2, "Home")
		l.RawSetInt(-2, count)
	}
}

func getAttribute[T any](obj map[string]any, name string) (T, error) {
	var v T
	attrib, ok := obj[name]
	if !ok {
		return v, errors.New("not found")
	}
	if v, ok = attrib.(T); !ok {
		return v, fmt.Errorf("invalid type: %T", attrib)
	}
	return v, nil
}
