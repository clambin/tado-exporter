package rules

import (
	"embed"
	"errors"
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/clambin/tado-exporter/internal/controller/rules/luart"
	"iter"
)

type luaScript struct {
	*lua.State
}

func loadLuaScript(name string, cfg ScriptConfig, fs *embed.FS) (luaScript, error) {
	r, err := cfg.script(fs)
	if err != nil {
		return luaScript{}, err
	}
	defer func() { _ = r.Close() }()

	var s luaScript
	s.State, err = luart.Compile(name, r)
	return s, err
}

func (l luaScript) pushHomeState(homeState HomeState) {
	luaObject := map[string]any{
		"Overlay": homeState.Overlay,
		"Home":    homeState.Home,
	}
	luart.PushMap(l.State, luaObject)
}

func (l luaScript) getHomeState(index int) (HomeState, error) {
	if !l.State.IsTable(index) {
		return HomeState{}, &errLuaInvalidResponse{fmt.Errorf("no table found at index %d", index)}
	}
	// more idiomatic for TableToMap to push these on the stack and then pop them with ToBoolean?
	obj := luart.TableToMap(l.State, l.State.AbsIndex(index))
	var hs HomeState
	var err error
	if hs.Overlay, err = getTableAttribute[bool](obj, "Overlay"); err != nil {
		return HomeState{}, &errLuaInvalidResponse{fmt.Errorf("homeState.Overlay: %w", err)}
	}
	if hs.Home, err = getTableAttribute[bool](obj, "Home"); err != nil {
		return HomeState{}, &errLuaInvalidResponse{err: fmt.Errorf("homeState.Home: %w", err)}
	}
	return hs, nil
}

func (l luaScript) pushZoneState(zoneState ZoneState) {
	luaObject := map[string]any{
		"Overlay": zoneState.Overlay,
		"Heating": zoneState.Heating,
	}
	luart.PushMap(l.State, luaObject)
}

func (l luaScript) getZoneState(index int) (ZoneState, error) {
	if !l.State.IsTable(index) {
		return ZoneState{}, &errLuaInvalidResponse{fmt.Errorf("no table found at index %d", index)}
	}
	// more idiomatic for TableToMap to push these on the stack and then pop them with ToBoolean?
	obj := luart.TableToMap(l.State, l.State.AbsIndex(index))
	var zs ZoneState
	var err error
	if zs.Overlay, err = getTableAttribute[bool](obj, "Overlay"); err != nil {
		return ZoneState{}, &errLuaInvalidResponse{fmt.Errorf("zoneState.Overlay: %w", err)}
	}
	if zs.Heating, err = getTableAttribute[bool](obj, "Heating"); err != nil {
		return ZoneState{}, &errLuaInvalidResponse{err: fmt.Errorf("zoneState.Home: %w", err)}
	}
	return zs, nil
}

func (l luaScript) pushDevices(devices iter.Seq[Device]) {
	l.State.NewTable()
	var count int
	for d := range devices {
		count++
		l.State.NewTable()
		l.State.PushString(d.Name) // Push the value for "Name"
		l.State.SetField(-2, "Name")
		l.State.PushBoolean(d.Home) // Push the value for "AtHome"
		l.State.SetField(-2, "Home")
		l.State.RawSetInt(-2, count)
	}
}

func getTableAttribute[T any](obj map[string]any, name string) (T, error) {
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
