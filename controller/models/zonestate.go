package models

import (
	"fmt"
	"github.com/clambin/tado"
)

type ZoneStateEnum int

const (
	ZoneUnknown ZoneStateEnum = iota
	ZoneOff     ZoneStateEnum = iota
	ZoneAuto    ZoneStateEnum = iota
	ZoneManual  ZoneStateEnum = iota
)

type ZoneState struct {
	State       ZoneStateEnum
	Temperature tado.Temperature
}

func (a ZoneState) String() string {
	switch a.State {
	case ZoneOff:
		return "off"
	case ZoneAuto:
		return "auto"
	case ZoneManual:
		return fmt.Sprintf("manual (%.1fºC)", a.Temperature.Celsius)
	}
	return "unknown"
}

func (a ZoneState) Equals(b ZoneState) bool {
	return a.State == b.State && a.Temperature.Celsius == b.Temperature.Celsius
}

func GetZoneState(zoneInfo tado.ZoneInfo) (state ZoneState) {
	if zoneInfo.Overlay.Type == "MANUAL" &&
		zoneInfo.Overlay.Setting.Type == "HEATING" &&
		zoneInfo.Overlay.Termination.Type == "MANUAL" {
		if zoneInfo.Overlay.Setting.Temperature.Celsius <= 5.0 {
			// TODO: probably more states that should be considered "off"?
			state.State = ZoneOff
		} else {
			state.State = ZoneManual
			state.Temperature = zoneInfo.Overlay.Setting.Temperature
		}
	} else {
		state.State = ZoneAuto
	}
	return
}
