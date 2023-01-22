package poller

import "github.com/clambin/tado"

type Update struct {
	Zones       map[int]tado.Zone
	ZoneInfo    map[int]tado.ZoneInfo
	UserInfo    map[int]tado.MobileDevice
	WeatherInfo tado.WeatherInfo
}

func (update Update) GetZoneID(name string) (int, bool) {
	for zoneID, zone := range update.Zones {
		if zone.Name == name {
			return zoneID, true
		}
	}
	return 0, false
}

func (update Update) GetUserID(name string) (int, bool) {
	for userID, user := range update.UserInfo {
		if user.Name == name {
			return userID, true
		}
	}
	return 0, false
}
