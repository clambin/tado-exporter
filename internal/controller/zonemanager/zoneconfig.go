package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
)

// TODO: move this to models?
func (mgr *Manager) makeZoneConfig(config []configuration.ZoneConfig) (zoneConfig map[int]model.ZoneConfig, err error) {
	var allZones, allUsers map[int]string

	allZones, err = mgr.getAllZones()

	if err == nil {
		allUsers, err = mgr.getAllUsers()
	}

	if err == nil {
		zoneConfig = make(map[int]model.ZoneConfig)

		for _, entry := range config {
			var zoneID int
			var ok bool

			if zoneID, ok = lookup(allZones, entry.ZoneID, entry.ZoneName); ok == false {
				log.WithFields(log.Fields{"id": entry.ZoneID, "name": entry.ZoneName}).Warning("ignoring invalid zone in configuration")
				continue
			}

			zoneConfigEntry := model.ZoneConfig{}

			if entry.AutoAway.Enabled {
				zoneConfigEntry.AutoAway.Enabled = true
				zoneConfigEntry.AutoAway.Delay = entry.AutoAway.Delay

				for _, user := range entry.AutoAway.Users {
					var userID int
					if userID, ok = lookup(allUsers, user.MobileDeviceID, user.MobileDeviceName); ok == false {
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

			zoneConfig[zoneID] = zoneConfigEntry
		}
	}
	return
}

func lookup(table map[int]string, id int, name string) (int, bool) {
	if _, ok := table[id]; ok == true {
		return id, true
	}
	for entryID, entryName := range table {
		if entryName == name {
			return entryID, true
		}
	}
	return 0, false
}

func (mgr *Manager) getAllZones() (zones map[int]string, err error) {
	var tadoZones []tado.Zone

	if tadoZones, err = mgr.API.GetZones(); err == nil {
		zones = make(map[int]string)

		for _, tadoZone := range tadoZones {
			zones[tadoZone.ID] = tadoZone.Name
		}
	}
	return
}

func (mgr *Manager) getAllUsers() (users map[int]string, err error) {
	var tadoUsers []tado.MobileDevice

	if tadoUsers, err = mgr.API.GetMobileDevices(); err == nil {
		users = make(map[int]string)

		for _, tadoUser := range tadoUsers {
			users[tadoUser.ID] = tadoUser.Name
		}
	}
	return
}
