package processor

import (
	"github.com/clambin/tado-exporter/poller"
	log "github.com/sirupsen/logrus"
)

func (server *Server) load(update *poller.Update) {
	server.zoneRules = make(map[int]ZoneRules)

	for _, entry := range server.zoneConfig {
		zoneID, _, found := update.LookupZone(entry.ZoneID, entry.ZoneName)

		if !found {
			log.WithFields(log.Fields{"id": entry.ZoneID, "name": entry.ZoneName}).Warning("ignoring invalid zone in configuration")
			continue
		}

		zoneConfigEntry := ZoneRules{}

		if entry.AutoAway.Enabled {
			zoneConfigEntry.AutoAway.Enabled = true
			zoneConfigEntry.AutoAway.Delay = entry.AutoAway.Delay

			for _, user := range entry.AutoAway.Users {
				var userID int
				userID, _, found = update.LookupUser(user.MobileDeviceID, user.MobileDeviceName)

				if !found {
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

		server.zoneRules[zoneID] = zoneConfigEntry
	}
}
