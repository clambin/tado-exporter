package tmp

import (
	"errors"
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"log/slog"
)

type state interface {
	Equals(o state) bool
	GetState() (bool, bool)
	Description() string
	LogValue() slog.Value
	ToLua(*lua.State)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ state = homeState{}

var homeStateString = map[bool]string{
	true:  "HOME",
	false: "AWAY",
}

type homeState struct {
	Home   bool
	Manual bool
}

func (s homeState) Equals(o state) bool {
	hs, ok := o.(homeState)
	return ok && s.Home == hs.Home && s.Manual == hs.Manual
}

func (s homeState) GetState() (bool, bool) {
	return s.Home, s.Manual
}

func (s homeState) Description() string {
	description := "setting home to " + homeStateString[s.Home] + " mode"
	if s.Manual {
		description += " (manual)"
	}
	return description
}

func (s homeState) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("home", s.Home),
		slog.Bool("manual", s.Manual),
	)
}

func (s homeState) ToLua(l *lua.State) {
	luaObject := map[string]any{
		"Home":   s.Home,
		"Manual": s.Manual,
	}
	luart.PushMap(l, luaObject)

}

func toHomeState(l *lua.State, index int) (homeState, error) {
	if !l.IsTable(index) {
		return homeState{}, fmt.Errorf("no table found at index %d", index)
	}
	// more idiomatic for TableToMap to push these on the stack and then pop them with ToBoolean?
	obj := luart.TableToMap(l, l.AbsIndex(index))
	home, ok := obj["Home"]
	if !ok {
		return homeState{}, errors.New("missing Home")
	}
	manual, ok := obj["Manual"]
	if !ok {
		return homeState{}, errors.New("missing Manual")
	}
	var s homeState
	if s.Home, ok = home.(bool); !ok {
		return homeState{}, errors.New("home is not a boolean")
	}
	if s.Manual, ok = manual.(bool); !ok {
		return homeState{}, errors.New("manual is not a boolean")
	}
	return s, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ state = zoneState{}

type zoneState struct {
	Heating bool
	Manual  bool
}

func (s zoneState) Equals(o state) bool {
	zs, ok := o.(zoneState)
	return ok && s.Heating == zs.Heating && s.Manual == zs.Manual
}

func (s zoneState) GetState() (bool, bool) {
	return s.Heating, s.Manual
}

var zoneStateString = map[bool]string{
	true:  "on",
	false: "off",
}

func (s zoneState) Description() string {
	description := "switching heating " + zoneStateString[s.Heating]
	if s.Manual {
		description += " (manual)"
	}
	return description
}

func (s zoneState) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("heating", s.Heating),
		slog.Bool("manual", s.Manual),
	)
}

func (s zoneState) ToLua(l *lua.State) {
	luaObject := map[string]any{
		"Heating": s.Heating,
		"Manual":  s.Manual,
	}
	luart.PushMap(l, luaObject)
}

func toZoneState(l *lua.State, index int) (zoneState, error) {
	if !l.IsTable(index) {
		return zoneState{}, fmt.Errorf("no table found at index %d", index)
	}

	obj := luart.TableToMap(l, l.AbsIndex(index))
	heating, ok := obj["Heating"]
	if !ok {
		return zoneState{}, errors.New("missing Heating")
	}
	manual, ok := obj["Manual"]
	if !ok {
		return zoneState{}, errors.New("missing Manual")
	}
	var s zoneState
	if s.Heating, ok = heating.(bool); !ok {
		return zoneState{}, errors.New("heating is not a boolean")
	}
	if s.Manual, ok = manual.(bool); !ok {
		return zoneState{}, errors.New("manual is not a boolean")
	}
	return s, nil
}
