package statemanager

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/models"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	tado.API
	zoneRules     map[int]models.ZoneRules
	zoneConfig    []configuration.ZoneConfig
	zoneNameCache map[int]string
	userNameCache map[int]string
}

func New(API tado.API, config []configuration.ZoneConfig) (mgr *Manager, err error) {
	mgr = &Manager{
		API:        API,
		zoneConfig: config,
	}
	err = mgr.initialize(context.Background())

	return
}

func (mgr *Manager) IsValidZoneID(zoneID int) (found bool) {
	_, found = mgr.zoneRules[zoneID]
	return
}

func (mgr *Manager) initialize(ctx context.Context) (err error) {
	if err = mgr.buildNameCache(ctx); err != nil {
		return
	}

	return mgr.load(mgr.zoneConfig)
}

func (mgr *Manager) load(config []configuration.ZoneConfig) (err error) {
	mgr.zoneRules = make(map[int]models.ZoneRules)

	for _, entry := range config {
		var zoneID int
		var ok bool

		zoneID, _, ok = mgr.LookupZone(entry.ZoneID, entry.ZoneName)

		if ok == false {
			log.WithFields(log.Fields{"id": entry.ZoneID, "name": entry.ZoneName}).Warning("ignoring invalid zone in configuration")
			continue
		}

		zoneConfigEntry := models.ZoneRules{}

		if entry.AutoAway.Enabled {
			zoneConfigEntry.AutoAway.Enabled = true
			zoneConfigEntry.AutoAway.Delay = entry.AutoAway.Delay

			for _, user := range entry.AutoAway.Users {
				var userID int
				userID, _, ok = mgr.LookupUser(user.MobileDeviceID, user.MobileDeviceName)

				if ok == false {
					log.WithFields(log.Fields{"id": user.MobileDeviceID, "name": user.MobileDeviceName}).Warning("ignoring invalid user in configuration")
					continue
				}
				zoneConfigEntry.AutoAway.Users = append(zoneConfigEntry.AutoAway.Users, userID)
			}

		}

		if entry.LimitOverlay.Enabled {
			zoneConfigEntry.LimitOverlay.Enabled = true
			zoneConfigEntry.LimitOverlay.Delay = entry.LimitOverlay.Delay
		}

		if entry.NightTime.Enabled {
			zoneConfigEntry.NightTime.Enabled = true
			zoneConfigEntry.NightTime.Time.Hour = entry.NightTime.Time.Hour
			zoneConfigEntry.NightTime.Time.Minutes = entry.NightTime.Time.Minutes
		}

		mgr.zoneRules[zoneID] = zoneConfigEntry
	}

	return
}

func (mgr *Manager) buildNameCache(ctx context.Context) (err error) {
	if err = mgr.buildZoneNameCache(ctx); err != nil {
		return
	}
	return mgr.buildUserNameCache(ctx)
}

func (mgr *Manager) buildZoneNameCache(ctx context.Context) (err error) {
	var zones []tado.Zone
	zones, err = mgr.API.GetZones(ctx)

	if err != nil {
		return
	}

	mgr.zoneNameCache = make(map[int]string)
	for _, zone := range zones {
		mgr.zoneNameCache[zone.ID] = zone.Name
	}

	return
}

func (mgr *Manager) buildUserNameCache(ctx context.Context) (err error) {
	var mobileDevices []tado.MobileDevice
	mobileDevices, err = mgr.API.GetMobileDevices(ctx)

	if err != nil {
		return
	}

	mgr.userNameCache = make(map[int]string)
	for _, mobileDevice := range mobileDevices {
		mgr.userNameCache[mobileDevice.ID] = mobileDevice.Name
	}

	return
}

func (mgr *Manager) LookupZone(id int, name string) (zoneID int, zoneName string, found bool) {
	for zoneID, zoneName = range mgr.zoneNameCache {
		if id == zoneID || name == zoneName {
			found = true
			return
		}
	}
	return
}

func (mgr *Manager) LookupUser(id int, name string) (userID int, userName string, found bool) {
	for userID, userName = range mgr.zoneNameCache {
		if id == userID || name == userName {
			found = true
			return
		}
	}
	return
}
