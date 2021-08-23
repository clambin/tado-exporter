package statemanager

import (
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/cache"
	"github.com/clambin/tado-exporter/controller/models"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	ZoneConfig []configuration.ZoneConfig
	Cache      *cache.Cache
	zoneRules  map[int]models.ZoneRules
}

func (mgr *Manager) initialize() {
	if mgr.zoneRules == nil {
		mgr.load()
	}
}

func (mgr *Manager) load() {
	mgr.zoneRules = make(map[int]models.ZoneRules)

	for _, entry := range mgr.ZoneConfig {
		var zoneID int
		var ok bool

		zoneID, _, ok = mgr.Cache.LookupZone(entry.ZoneID, entry.ZoneName)

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
				userID, _, ok = mgr.Cache.LookupUser(user.MobileDeviceID, user.MobileDeviceName)

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
