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

func (a ZoneState) String() string {
	switch a.State {
	case Off:
		return "off"
	case Auto:
		return "auto"
	case Manual:
		return fmt.Sprintf("manual (%.1fºC)", a.Temperature.Celsius)
	}
	return "unknown"
}

func (a ZoneState) Equals(b ZoneState) bool {
	return a.State == b.State && a.Temperature.Celsius == b.Temperature.Celsius
}
