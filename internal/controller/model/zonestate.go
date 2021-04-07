package model

import (
	"fmt"
	"github.com/clambin/tado-exporter/pkg/tado"
)

type ZoneStateEnum int

const (
	Off    ZoneStateEnum = 0
	Auto   ZoneStateEnum = 1
	Manual ZoneStateEnum = 2
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
	default:
		return "unknown"
	}
}
