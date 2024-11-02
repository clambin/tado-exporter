package poller

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado/v2"
)

type Update struct {
	tado.HomeBase
	tado.HomeState
	tado.Weather
	Zones
	MobileDevices
}

func (u Update) Home() bool {
	return *u.HomeState.Presence == tado.HOME
}

type Zone struct {
	tado.Zone
	tado.ZoneState
}

type Zones []Zone

func (z Zones) GetZone(name string) (Zone, error) {
	for _, zone := range z {
		if *zone.Name == name {
			return zone, nil
		}
	}
	return Zone{}, fmt.Errorf("invalid zone: %q", name)
}

const ZoneOverlayTerminationTypeNONE = tado.ZoneOverlayTerminationType("NONE")

func (z Zone) GetZoneOverlayTerminationType() tado.ZoneOverlayTerminationType {
	if z.ZoneState.Overlay == nil {
		return ZoneOverlayTerminationTypeNONE
	}
	return *z.ZoneState.Overlay.Termination.Type
}

func (z Zone) GetTargetTemperature() float32 {
	if z.ZoneState.Setting.Temperature == nil {
		return 0
	}
	return *z.ZoneState.Setting.Temperature.Celsius
}

type MobileDevices []tado.MobileDevice

func (m MobileDevices) GetMobileDevice(deviceName string) (tado.MobileDevice, bool) {
	for _, device := range m {
		if *device.Name == deviceName && *device.Settings.GeoTrackingEnabled {
			return device, true
		}
	}
	return tado.MobileDevice{}, false
}

func (m MobileDevices) GetDeviceState(ids ...tado.MobileDeviceId) ([]string, []string) {
	lookup := make(set.Set[tado.MobileDeviceId], len(ids))
	for _, id := range ids {
		lookup.Add(id)
	}

	home := make([]string, 0, len(m))
	away := make([]string, 0, len(m))

	for _, device := range m {
		if (len(ids) == 0 || lookup.Contains(*device.Id)) && *device.Settings.GeoTrackingEnabled {
			if *device.Location.AtHome {
				home = append(home, *device.Name)
			} else {
				away = append(away, *device.Name)
			}
		}
	}
	return home, away
}
