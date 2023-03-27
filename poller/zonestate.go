package poller

import "github.com/clambin/tado"

// ZoneState is the state of the zone, i.e. heating is off, controlled automatically, or controlled manually
type ZoneState int

const (
	// ZoneStateUnknown indicates the zone's state is not initialized yet
	ZoneStateUnknown ZoneState = iota
	// ZoneStateOff indicates the zone's heating is switched off
	ZoneStateOff
	// ZoneStateAuto indicates the zone's heating is controlled manually (e.g. as per schedule)
	ZoneStateAuto
	// ZoneStateTemporaryManual indicates the zone's target temperature is set manually, for a period of time
	ZoneStateTemporaryManual
	// ZoneStateManual indicates the zone's target temperature is set manually
	ZoneStateManual
)

// String returns a string representation of a ZoneState
func (s ZoneState) String() string {
	names := map[ZoneState]string{
		ZoneStateUnknown:         "unknown",
		ZoneStateOff:             "off",
		ZoneStateAuto:            "auto",
		ZoneStateTemporaryManual: "manual (temp)",
		ZoneStateManual:          "manual",
	}
	name, ok := names[s]
	if !ok {
		name = "(invalid)"
	}
	return name
}

// GetZoneState returns the state of the zone
func GetZoneState(z tado.ZoneInfo) ZoneState {
	if z.Setting.Power == "OFF" {
		return ZoneStateOff
	}
	switch z.Overlay.GetMode() {
	case tado.NoOverlay:
		return ZoneStateAuto
	case tado.PermanentOverlay:
		return ZoneStateManual
	case tado.TimerOverlay, tado.NextBlockOverlay:
		return ZoneStateTemporaryManual
	}
	return ZoneStateUnknown
}
