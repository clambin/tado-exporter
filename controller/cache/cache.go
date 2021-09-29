package cache

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"sync"
	"time"
)

type Cache struct {
	update *poller.Update
	lock   sync.RWMutex
}

func (cache *Cache) Update(update *poller.Update) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.update = update
}

func (cache *Cache) GetZoneName(id int) (name string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var zone tado.Zone
	zone, ok = cache.update.Zones[id]
	if ok {
		name = zone.Name
	} else {
		name = "unknown"
	}
	return
}

func (cache *Cache) GetUserName(id int) (name string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var device tado.MobileDevice
	device, ok = cache.update.UserInfo[id]

	if ok {
		name = device.Name
	} else {
		name = "unknown"
	}
	return
}

func (cache *Cache) LookupZone(id int, name string) (foundID int, foundName string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return cache.update.LookupZone(id, name)
}

func (cache *Cache) LookupUser(id int, name string) (foundID int, foundName string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return cache.update.LookupUser(id, name)
}

func (cache *Cache) GetZones() (zoneIDs []int) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	for zoneID := range cache.update.Zones {
		zoneIDs = append(zoneIDs, zoneID)
	}
	return
}

func (cache *Cache) GetZoneInfo(id int) (temperature, targetTemperature float64, zoneState tado.ZoneState, duration time.Duration, found bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var zoneInfo tado.ZoneInfo
	if zoneInfo, found = cache.update.ZoneInfo[id]; found {
		temperature = zoneInfo.SensorDataPoints.Temperature.Celsius
		zoneState = zoneInfo.GetState()
		if zoneState == tado.ZoneStateAuto {
			targetTemperature = zoneInfo.Setting.Temperature.Celsius

			// expired overlay may still be in the cache
			if targetTemperature == 0 {
				targetTemperature = zoneInfo.Overlay.Setting.Temperature.Celsius
			}
		} else {
			targetTemperature = zoneInfo.Overlay.Setting.Temperature.Celsius
			if zoneState == tado.ZoneStateTemporaryManual {
				duration = time.Duration(zoneInfo.Overlay.Termination.DurationInSeconds) * time.Second
			}
		}
	}
	return
}

func (cache *Cache) GetUsers() (userIDs []int) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	for userID := range cache.update.UserInfo {
		userIDs = append(userIDs, userID)
	}
	return
}

func (cache *Cache) GetUserInfo(userID int) (location tado.MobileDeviceLocationState, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var userInfo tado.MobileDevice
	userInfo, ok = cache.update.UserInfo[userID]
	if ok {
		location = userInfo.IsHome()
	}
	return
}
