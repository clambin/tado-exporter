// tadoproxy caches all tado-related state, so tado-controller doesn't need to repeatedly query tado.com
package tadoproxy

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
)

// Proxy structure is used to call Tado APIs and caches all tado-related state
type Proxy struct {
	tado.API

	Zone         map[int]*tado.Zone
	ZoneInfo     map[int]*tado.ZoneInfo
	MobileDevice map[int]*tado.MobileDevice
}

// Refresh the Cache
func (proxy *Proxy) Refresh() (err error) {
	var (
		zones         []*tado.Zone
		zoneInfo      *tado.ZoneInfo
		mobileDevices []*tado.MobileDevice
	)

	zoneMap := make(map[int]*tado.Zone)
	if zones, err = proxy.GetZones(); err == nil {
		for _, zone := range zones {
			zoneMap[zone.ID] = zone
		}
	}
	proxy.Zone = zoneMap

	zoneInfoMap := make(map[int]*tado.ZoneInfo)
	for zoneID := range proxy.Zone {
		if zoneInfo, err = proxy.GetZoneInfo(zoneID); err == nil {
			zoneInfoMap[zoneID] = zoneInfo
		}
	}
	proxy.ZoneInfo = zoneInfoMap

	mobileDeviceMap := make(map[int]*tado.MobileDevice)
	if mobileDevices, err = proxy.GetMobileDevices(); err == nil {
		for _, mobileDevice := range mobileDevices {
			mobileDeviceMap[mobileDevice.ID] = mobileDevice
		}
	}
	proxy.MobileDevice = mobileDeviceMap

	log.WithFields(log.Fields{
		"err":           err,
		"zones":         len(proxy.Zone),
		"zoneInfos":     len(proxy.ZoneInfo),
		"mobileDevices": len(proxy.MobileDevice),
	}).Debug("updateTadoConfig")

	return
}
