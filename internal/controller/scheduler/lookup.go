package scheduler

import (
	"github.com/clambin/tado-exporter/pkg/tado"
)

// LookupMobileDevice returns the mobile device matching the mobileDeviceID or mobileDeviceName from the list of mobile devices
func LookupMobileDevice(tadoData *TadoData, mobileDeviceID int, mobileDeviceName string) *tado.MobileDevice {
	for id, mobileDevice := range tadoData.MobileDevice {
		if id == mobileDeviceID || mobileDevice.Name == mobileDeviceName {
			return &mobileDevice
		}
	}
	return nil
}

// LookupZone returns the zone matching zoneID or zoneName from the list of zones
func LookupZone(tadoData *TadoData, zoneID int, zoneName string) *tado.Zone {
	for id, zone := range tadoData.Zone {
		if id == zoneID || zone.Name == zoneName {
			return &zone
		}
	}
	return nil
}
