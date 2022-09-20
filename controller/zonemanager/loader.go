package zonemanager

import (
	"fmt"
	"github.com/clambin/tado-exporter/poller"
)

func (m *Manager) load(update *poller.Update) (err error) {
	if m.loaded {
		return
	}

	zoneID, zoneName, found := update.LookupZone(m.config.ZoneID, m.config.ZoneName)
	if !found {
		return fmt.Errorf("invalid zone found in config file: zoneID: %d, zoneName: %s", m.config.ZoneID, m.config.ZoneName)
	}

	m.config.ZoneID = zoneID
	m.config.ZoneName = zoneName

	for index, user := range m.config.AutoAway.Users {
		var userID int
		var userName string
		userID, userName, found = update.LookupUser(user.MobileDeviceID, user.MobileDeviceName)

		if !found {
			return fmt.Errorf("invalid user found in config file: zoneID: %d, zoneName: %s", user.MobileDeviceID, user.MobileDeviceName)
		}

		m.config.AutoAway.Users[index].MobileDeviceID = userID
		m.config.AutoAway.Users[index].MobileDeviceName = userName
	}

	m.loaded = true
	return nil
}
