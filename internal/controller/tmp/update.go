package tmp

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
)

type update struct {
	ZoneStates map[string]zoneInfo
	devices
	tado.HomeId
	homeState homeState
}

type zoneInfo struct {
	tado.ZoneId
	zoneState
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
	return homeState{
		Home:   *u.HomeState.Presence == tado.HOME,
		Manual: u.HomeState.PresenceLocked != nil && *u.HomeState.PresenceLocked,
	}
}

func zoneStateFromPollerZone(z poller.Zone) zoneState {
	return zoneState{
		Heating: *z.ZoneState.Setting.Power == tado.PowerON,
		Manual:  z.ZoneState.Overlay != nil && *z.ZoneState.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeMANUAL,
	}
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
