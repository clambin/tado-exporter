package controller

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
)

type update struct {
	homeState
	tado.HomeId
	ZoneStates map[string]zoneInfo
	devices
}

type zoneInfo struct {
	zoneState
	tado.ZoneId
}

func updateFromPollerUpdate(u poller.Update) update {
	h := update{
		HomeId:    *u.HomeBase.Id,
		homeState: homeStateFromPollerZone(u),
	}

	h.ZoneStates = make(map[string]zoneInfo, len(u.Zones))
	for _, zone := range u.Zones {
		if *zone.ZoneState.Setting.Type != tado.HEATING {
			continue
		}

		h.ZoneStates[*zone.Name] = zoneInfo{
			ZoneId:    *zone.Id,
			zoneState: zoneStateFromPollerZone(zone),
		}
	}

	h.devices = make(devices, 0, len(u.MobileDevices))
	for d := range u.GeoTrackedDevices() {
		h.devices = append(h.devices, device{Name: *d.Name, Home: *d.Location.AtHome})
	}

	return h
}

func homeStateFromPollerZone(u poller.Update) homeState {
	if u.HomeState.PresenceLocked == nil || !*u.HomeState.PresenceLocked {
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

func zoneStateFromPollerZone(z poller.Zone) zoneState {
	if z.ZoneState.Overlay == nil || *z.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeTIMER {
		return ZoneStateAuto
	}

	// the ZoneStateOff state allows us to switch off heating.  But do we need it when reading the update?
	if *z.ZoneState.Overlay.Setting.Power == tado.PowerOFF {
		return ZoneStateOff
	}
	return ZoneStateManual
}

func (u update) GetHomeState() homeState {
	return u.homeState
}

func (u update) GetZoneState(name string) (zoneState, bool) {
	z, ok := u.ZoneStates[name]
	return z.zoneState, ok
}

func (u update) GetDevices() devices {
	return u.devices
}
