package cache

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"sync"
)

type Cache struct {
	zones     map[int]tado.Zone
	zoneInfos map[int]tado.ZoneInfo
	users     map[int]tado.MobileDevice
	lock      sync.RWMutex
}

func New() *Cache {
	return &Cache{}
}

func (cache *Cache) Update(update *poller.Update) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.zones = update.Zones
	cache.zoneInfos = update.ZoneInfo
	cache.users = update.UserInfo
}

func (cache *Cache) GetZoneName(id int) (name string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var zone tado.Zone
	zone, ok = cache.zones[id]
	if ok {
		name = zone.Name
	}
	return
}

func (cache *Cache) GetUserName(id int) (name string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var device tado.MobileDevice
	device, ok = cache.users[id]

	if ok {
		name = device.Name
	}
	return
}

func (cache *Cache) LookupZone(id int, name string) (foundID int, foundName string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var zone tado.Zone
	for foundID, zone = range cache.zones {
		if foundID == id || zone.Name == name {
			return foundID, zone.Name, true
		}
	}
	return
}

func (cache *Cache) LookupUser(id int, name string) (foundID int, foundName string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var device tado.MobileDevice
	for foundID, device = range cache.users {
		if foundID == id || device.Name == name {
			return foundID, device.Name, true
		}
	}
	return
}

func (cache *Cache) GetZones() (zoneIDs []int) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	for zoneID := range cache.zones {
		zoneIDs = append(zoneIDs, zoneID)
	}
	return
}

func (cache *Cache) GetZoneInfo(id int) (temperature, targetTemperature float64, zoneState tado.ZoneState, found bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var zoneInfo tado.ZoneInfo
	if zoneInfo, found = cache.zoneInfos[id]; found {
		temperature = zoneInfo.SensorDataPoints.Temperature.Celsius
		zoneState = zoneInfo.GetState()
		if zoneState == tado.ZoneStateAuto {
			targetTemperature = zoneInfo.Setting.Temperature.Celsius
		} else {
			targetTemperature = zoneInfo.Overlay.Setting.Temperature.Celsius
		}
		return
	}

	return
}
