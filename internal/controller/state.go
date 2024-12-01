package controller

import (
	"log/slog"
)

type state interface {
	Equals(o state) bool
	Overlay() bool
	Mode() bool
	String() string
	slog.LogValuer
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ state = homeState{}

var homeStateString = map[bool]string{
	true:  "HOME",
	false: "AWAY",
}

type homeState struct {
	overlay bool
	home    bool
}

func (s homeState) Equals(o state) bool {
	hs, ok := o.(homeState)
	return ok && s.home == hs.home && s.overlay == hs.overlay
}

func (s homeState) Overlay() bool {
	return s.overlay
}

func (s homeState) Mode() bool {
	return s.home
}

func (s homeState) String() string {
	description := homeStateString[s.home] + " mode"
	if s.overlay {
		description += " (manual)"
	}
	return description
}

func (s homeState) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("overlay", s.overlay),
		slog.Bool("home", s.home),
	)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ state = zoneState{}

type zoneState struct {
	overlay bool
	heating bool
}

func (s zoneState) Equals(o state) bool {
	zs, ok := o.(zoneState)
	return ok && s.heating == zs.heating && s.overlay == zs.overlay
}

func (s zoneState) Overlay() bool {
	return s.overlay
}

func (s zoneState) Mode() bool {
	return s.heating
}

var zoneStateString = map[bool]string{
	true:  "on",
	false: "off",
}

func (s zoneState) String() string {
	description := zoneStateString[s.heating]
	if s.overlay {
		description += " (manual)"
	}
	return description
}

func (s zoneState) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("overlay", s.overlay),
		slog.Bool("heating", s.heating),
	)
}
