package scheduler

import (
	"github.com/clambin/tado-exporter/pkg/tado"
)

// TadoData contains all data retrieved from Tado and needed to evaluate rules.
//
// This allows data to be shared between the scheduler and its clients without locking mechanisms: each client
// gets its own copy of the data without having to worry if its changed by a subsequent refresh
type TadoData struct {
	Zone         map[int]tado.Zone
	ZoneInfo     map[int]tado.ZoneInfo
	MobileDevice map[int]tado.MobileDevice
}

// LookupMobileDevice returns the mobile device matching the mobileDeviceID or mobileDeviceName from the list of mobile devices
func (tadoData *TadoData) LookupMobileDevice(mobileDeviceID int, mobileDeviceName string) *tado.MobileDevice {
	for id, mobileDevice := range tadoData.MobileDevice {
		if id == mobileDeviceID || mobileDevice.Name == mobileDeviceName {
			return &mobileDevice
		}
	}
	return nil
}

// LookupZone returns the zone matching zoneID or zoneName from the list of zones
func (tadoData *TadoData) LookupZone(zoneID int, zoneName string) *tado.Zone {
	for id, zone := range tadoData.Zone {
		if id == zoneID || zone.Name == zoneName {
			return &zone
		}
	}
	return nil
}
