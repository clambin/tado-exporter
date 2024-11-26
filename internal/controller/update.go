package controller

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
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
		HomeId:    *u.HomeBase.Id,
		HomeState: homeStateFromPollerZone(u),
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

func homeStateFromPollerZone(u poller.Update) HomeState {
	if u.HomeState.PresenceLocked == nil || *u.HomeState.PresenceLocked == false {
		return HomeStateAuto
	}
	switch *u.HomeState.Presence {
	case tado.HOME:
		return HomeStateHome
	case tado.AWAY:
		return HomeStateAway
	default:
		panic("unknown home state" + *u.HomeState.Presence)
	}
}

func zoneStateFromPollerZone(z poller.Zone) ZoneState {
	if z.ZoneState.Overlay == nil || *z.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeTIMER {
		return ZoneStateAuto
	}

	// the ZoneStateOff state allows us to switch off heating.  But do we need it when reading the update?
	if *z.ZoneState.Overlay.Setting.Power == tado.PowerOFF {
		return ZoneStateOff
	}
	return ZoneStateManual
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
