package testutils

import (
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
)

func Update(options ...UpdateOption) poller.Update {
	var u poller.Update
	for _, option := range options {
		option(&u)
	}
	return u
}

type UpdateOption func(*poller.Update)

func WithHome(id tado.HomeId, name string, presence tado.HomePresence) UpdateOption {
	return func(u *poller.Update) {
		u.HomeBase.Id = &id
		u.HomeBase.Name = &name
		u.HomeState.Presence = &presence
	}
}

func WithZone(id tado.ZoneId, name string, power tado.Power, temperature float32, insideTemperature float32, options ...ZoneOption) UpdateOption {
	return func(u *poller.Update) {
		zone := poller.Zone{
			Zone: tado.Zone{Id: &id, Name: &name},
			ZoneState: tado.ZoneState{
				Setting:          &tado.ZoneSetting{Type: oapi.VarP(tado.HEATING), Power: &power, Temperature: &tado.Temperature{Celsius: &temperature}},
				SensorDataPoints: &tado.SensorDataPoints{InsideTemperature: &tado.TemperatureDataPoint{Celsius: &insideTemperature}},
			},
		}
		for _, option := range options {
			option(&zone)
		}
		u.Zones = append(u.Zones, zone)
	}
}

type ZoneOption func(*poller.Zone)

func WithZoneOverlay(terminationType tado.ZoneOverlayTerminationType, remaining int) ZoneOption {
	return func(zone *poller.Zone) {
		zone.ZoneState.Overlay = &tado.ZoneOverlay{
			Termination: &tado.ZoneOverlayTermination{
				Type:                   &terminationType,
				RemainingTimeInSeconds: &remaining,
			},
		}
	}
}

func WithMobileDevice(id tado.MobileDeviceId, name string, options ...MobileDeviceOption) UpdateOption {
	return func(u *poller.Update) {
		m := tado.MobileDevice{
			Id:       &id,
			Name:     &name,
			Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(false)},
		}
		for _, option := range options {
			option(&m)
		}
		u.MobileDevices = append(u.MobileDevices, m)
	}
}

type MobileDeviceOption func(*tado.MobileDevice)

func WithGeoTracking() MobileDeviceOption {
	return func(m *tado.MobileDevice) {
		m.Settings.GeoTrackingEnabled = oapi.VarP(true)
	}
}

func WithLocation(home, stale bool) MobileDeviceOption {
	return func(m *tado.MobileDevice) {
		WithGeoTracking()(m)
		m.Location = &tado.MobileDeviceLocation{AtHome: &home, Stale: &stale}
	}
}
