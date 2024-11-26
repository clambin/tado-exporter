package controller

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"strings"
)

type Update struct {
	HomeState
	tado.HomeId
	ZoneStates map[string]ZoneInfo
	Devices
}

type ZoneInfo struct {
	ZoneState
	tado.ZoneId
}

func updateFromPollerUpdate(u poller.Update) Update {
	h := Update{
		HomeState: HomeState(strings.ToLower(string(*u.HomeState.Presence))),
		HomeId:    *u.HomeBase.Id,
	}

	h.ZoneStates = make(map[string]ZoneInfo, len(u.Zones))
	for _, zone := range u.Zones {
		if *zone.ZoneState.Setting.Type != tado.HEATING {
			continue
		}

		h.ZoneStates[*zone.Name] = ZoneInfo{
			ZoneId:    *zone.Id,
			ZoneState: zoneStateFromPollerZone(zone),
		}
	}

	h.Devices = make(Devices, 0, len(u.MobileDevices))
	for device := range u.GeoTrackedDevices() {
		h.Devices = append(h.Devices, Device{Name: *device.Name, Home: *device.Location.AtHome})
	}

	return h
}

func zoneStateFromPollerZone(z poller.Zone) ZoneState {
	switch {
	case z.ZoneState.Overlay == nil:
		return ZoneStateAuto
	case *z.ZoneState.Setting.Power == tado.PowerOFF:
		return ZoneStateOff
	case *z.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeMANUAL:
		return ZoneStateManual
	case *z.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeTIMER:
		return ZoneStateAuto // we don't handle timer overlays; treat them as auto
	default:
		panic("unknown zone state")
	}
}

func (u Update) GetHomeState() HomeState {
	return u.HomeState
}

func (u Update) GetZoneState(name string) (ZoneState, bool) {
	z, ok := u.ZoneStates[name]
	return z.ZoneState, ok
}

func (u Update) GetDevices() Devices {
	return u.Devices
}
