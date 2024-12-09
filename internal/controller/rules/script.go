package rules

import (
	"embed"
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
	luart.LuaObject{
		"Overlay": homeState.Overlay,
		"Home":    homeState.Home,
	}.Push(l.State)
}

func (l luaScript) getHomeState(index int) (HomeState, error) {
	if !l.State.IsTable(index) {
		return HomeState{}, &errLuaInvalidResponse{fmt.Errorf("no table found at index %d", index)}
	}
	// more idiomatic for GetObject to push these on the stack and then pop them with ToBoolean?
	obj := luart.GetObject(l.State, l.State.AbsIndex(index))
	var hs HomeState
	var err error
	if hs.Overlay, err = luart.GetObjectAttribute[bool](obj, "Overlay"); err != nil {
		return HomeState{}, &errLuaInvalidResponse{fmt.Errorf("homeState.Overlay: %w", err)}
	}
	if hs.Home, err = luart.GetObjectAttribute[bool](obj, "Home"); err != nil {
		return HomeState{}, &errLuaInvalidResponse{err: fmt.Errorf("homeState.Home: %w", err)}
	}
	return hs, nil
}

func (l luaScript) pushZoneState(zoneState ZoneState) {
	luart.LuaObject{
		"Overlay": zoneState.Overlay,
		"Heating": zoneState.Heating,
	}.Push(l.State)
}

func (l luaScript) getZoneState(index int) (ZoneState, error) {
	if !l.State.IsTable(index) {
		return ZoneState{}, &errLuaInvalidResponse{fmt.Errorf("no table found at index %d", index)}
	}
	// more idiomatic for GetObject to push these on the stack and then pop them with ToBoolean?
	obj := luart.GetObject(l.State, l.State.AbsIndex(index))
	var zs ZoneState
	var err error
	if zs.Overlay, err = luart.GetObjectAttribute[bool](obj, "Overlay"); err != nil {
		return ZoneState{}, &errLuaInvalidResponse{fmt.Errorf("zoneState.Overlay: %w", err)}
	}
	if zs.Heating, err = luart.GetObjectAttribute[bool](obj, "Heating"); err != nil {
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

func (l luaScript) pushArgs(args Args) {
	luart.LuaObject(args).Push(l.State)
}
