package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	log "github.com/sirupsen/logrus"
)

func makeZoneConfig(config []configuration.ZoneConfig, allZones, allUsers map[int]string) (zoneConfig map[int]model.ZoneConfig) {
	zoneConfig = make(map[int]model.ZoneConfig)

	for _, entry := range config {
		var zoneID int
		var ok bool

		if zoneID, ok = lookup(allZones, entry.ZoneID, entry.ZoneName); ok == false {
			log.WithFields(log.Fields{"id": entry.ZoneID, "name": entry.ZoneName}).Warning("ignoring invalid zone in configuration")
			continue
		}

		zoneConfigEntry := model.ZoneConfig{Users: make([]int, 0)}

		for _, user := range entry.Users {
			var userID int

			if userID, ok = lookup(allUsers, user.MobileDeviceID, user.MobileDeviceName); ok == false {
				log.WithFields(log.Fields{"id": user.MobileDeviceID, "name": user.MobileDeviceName}).Warning("ignoring invalid user in configuration")
				continue
			}

			zoneConfigEntry.Users = append(zoneConfigEntry.Users, userID)
		}

		if entry.LimitOverlay.Enabled {
			zoneConfigEntry.LimitOverlay.Enabled = true
			zoneConfigEntry.LimitOverlay.Limit = entry.LimitOverlay.Limit
		}

		if entry.NightTime.Enabled {
			zoneConfigEntry.NightTime.Enabled = true
			zoneConfigEntry.NightTime.Time.Hour = entry.NightTime.Time.Hour
			zoneConfigEntry.NightTime.Time.Minutes = entry.NightTime.Time.Minutes
		}

		zoneConfig[zoneID] = zoneConfigEntry
	}
	return
}
