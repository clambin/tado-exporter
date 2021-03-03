package registry

import (
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
)

// Registry is the heart of the controller.  On run, it updates all info from Tado
// and signals any registered rules to run
type Registry struct {
	tado.API
	TadoBot  *tadobot.TadoBot
	clients  []chan *TadoData
	tadoData TadoData
}

// TadoData contains all data retrieved from Tado and needed to evaluate rules.
//
// This allows data to be shared between the registry and its clients without locking mechanisms: each client
// gets its own copy of the data without having to worry if its changed by a subsequent refresh
type TadoData struct {
	Zone         map[int]tado.Zone
	ZoneInfo     map[int]tado.ZoneInfo
	MobileDevice map[int]tado.MobileDevice
}

// Register a new client. This allocates a channel that the client should listen on for updates
func (registry *Registry) Register() (channel chan *TadoData) {
	if registry.clients == nil {
		registry.clients = make([]chan *TadoData, 0)
	}

	channel = make(chan *TadoData, 1)
	registry.clients = append(registry.clients, channel)
	return channel
}

// Run an update and signal any clients to evaluate their rules
func (registry *Registry) Run() (err error) {
	// refresh the cache
	if registry.tadoData, err = registry.refresh(); err == nil {
		// Notify all clients
		for _, client := range registry.clients {
			clientTadoData := registry.tadoData
			client <- &clientTadoData
		}
	}
	return
}

// Stop signals all clients to shut down
func (registry *Registry) Stop() {
	for _, client := range registry.clients {
		client <- nil
	}
}

// Refresh the Cache
func (registry *Registry) refresh() (tadoData TadoData, err error) {
	var (
		zones         []*tado.Zone
		zoneInfo      *tado.ZoneInfo
		mobileDevices []*tado.MobileDevice
	)

	zoneMap := make(map[int]tado.Zone)
	if zones, err = registry.GetZones(); err == nil {
		for _, zone := range zones {
			zoneMap[zone.ID] = *zone
		}
	}
	tadoData.Zone = zoneMap

	zoneInfoMap := make(map[int]tado.ZoneInfo)
	for zoneID := range tadoData.Zone {
		if zoneInfo, err = registry.GetZoneInfo(zoneID); err == nil {
			zoneInfoMap[zoneID] = *zoneInfo
		}
	}
	tadoData.ZoneInfo = zoneInfoMap

	mobileDeviceMap := make(map[int]tado.MobileDevice)
	if mobileDevices, err = registry.GetMobileDevices(); err == nil {
		for _, mobileDevice := range mobileDevices {
			mobileDeviceMap[mobileDevice.ID] = *mobileDevice
		}
	}
	tadoData.MobileDevice = mobileDeviceMap

	log.WithFields(log.Fields{
		"err":           err,
		"zones":         len(tadoData.Zone),
		"zoneInfos":     len(tadoData.ZoneInfo),
		"mobileDevices": len(tadoData.MobileDevice),
	}).Debug("updateTadoConfig")

	return
}

func (registry *Registry) Notify(title, message string) (err error) {
	if registry.TadoBot != nil {
		err = registry.TadoBot.SendMessage(title, message)
	}
	return
}
