package model

import (
	"fmt"
	"github.com/clambin/tado-exporter/pkg/tado"
)

type ZoneStateEnum int

const (
	Unknown ZoneStateEnum = 0
	Off     ZoneStateEnum = 1
	Auto    ZoneStateEnum = 2
	Manual  ZoneStateEnum = 3
)

type ZoneState struct {
	State       ZoneStateEnum
	Temperature tado.Temperature
}

func (state ZoneState) String() string {
	switch state.State {
	case Off:
		return "off"
	case Auto:
		return "auto"
	case Manual:
		return fmt.Sprintf("manual (%.1f)", state.Temperature.Celsius)
	}
	return "unknown"
}
