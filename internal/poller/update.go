package poller

import (
	"iter"
	"log/slog"

	"codeberg.org/clambin/go-common/set"
	"github.com/clambin/tado/v2"
)

type Update struct {
	HomeBase      tado.HomeBase
	HomeState     tado.HomeState
	Weather       tado.Weather
	Zones         Zones
	MobileDevices MobileDevices
}

func (u Update) Home() bool {
	return *u.HomeState.Presence == tado.HOME
}

type Zone struct {
	Zone      tado.Zone
	ZoneState tado.ZoneState
}

type Zones []Zone

func (z Zones) GetZone(name string) (Zone, bool) {
	for _, zone := range z {
		if *zone.Zone.Name == name {
			return zone, true
		}
	}
	return Zone{}, false
}

const ZoneOverlayTerminationTypeNONE = tado.ZoneOverlayTerminationType("NONE")

func (z Zone) GetZoneOverlayTerminationType() tado.ZoneOverlayTerminationType {
	if z.ZoneState.Overlay == nil {
		return ZoneOverlayTerminationTypeNONE
	}
	return *z.ZoneState.Overlay.Termination.Type
}

func (z Zone) GetTargetTemperature() float32 {
	if *z.ZoneState.Setting.Power != tado.PowerON {
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

func (m MobileDevices) GeoTrackedDevices() iter.Seq[tado.MobileDevice] {
	return func(yield func(tado.MobileDevice) bool) {
		for _, device := range m {
			if *device.Settings.GeoTrackingEnabled && device.Location != nil {
				if !yield(device) {
					return
				}
			}
		}
	}
}

func (m MobileDevices) LogValue() slog.Value {
	devices := make([]slog.Attr, 0, len(m))
	for device := range m.GeoTrackedDevices() {
		devices = append(devices, slog.Group(*device.Name,
			slog.Bool("geotracked", *device.Settings.GeoTrackingEnabled),
			slog.Bool("home", *device.Location.AtHome),
		))
	}
	return slog.AnyValue(devices)
}
