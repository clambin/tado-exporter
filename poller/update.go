package poller

import "github.com/clambin/tado"

type Update struct {
	Zones       map[int]tado.Zone
	ZoneInfo    map[int]tado.ZoneInfo
	UserInfo    map[int]tado.MobileDevice
	WeatherInfo tado.WeatherInfo
}

func (update *Update) LookupZone(id int, name string) (foundID int, foundName string, ok bool) {
	var zone tado.Zone
	for foundID, zone = range update.Zones {
		if foundID == id || zone.Name == name {
			return foundID, zone.Name, true
		}
	}
	return
}

func (update *Update) LookupUser(id int, name string) (foundID int, foundName string, ok bool) {
	var device tado.MobileDevice
	for foundID, device = range update.UserInfo {
		if foundID == id || device.Name == name {
			return foundID, device.Name, true
		}
	}
	return
}
