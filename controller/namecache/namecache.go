package namecache

import (
	"github.com/clambin/tado-exporter/poller"
	"sync"
)

type Cache struct {
	zones map[int]string
	users map[int]string
	lock  sync.RWMutex
}

func New() *Cache {
	return &Cache{
		zones: make(map[int]string),
		users: make(map[int]string),
	}
}

func (cache *Cache) Update(update *poller.Update) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	for zoneID, zone := range update.Zones {
		name, ok := cache.zones[zoneID]

		if ok == false || name != zone.Name {
			cache.zones[zoneID] = zone.Name
		}
	}
	for userID, user := range update.UserInfo {
		name, ok := cache.users[userID]

		if ok == false || name != user.Name {
			cache.users[userID] = user.Name
		}
	}
}

func (cache *Cache) GetZoneName(id int) (name string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	name, ok = cache.zones[id]
	return
}

func (cache *Cache) GetUserName(id int) (name string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	name, ok = cache.users[id]
	return
}

func (cache *Cache) LookupZone(id int, name string) (foundID int, foundName string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	for foundID, foundName = range cache.zones {
		if foundID == id || foundName == name {
			ok = true
			break
		}
	}
	return
}

func (cache *Cache) LookupUser(id int, name string) (foundID int, foundName string, ok bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	for foundID, foundName = range cache.users {
		if foundID == id || foundName == name {
			ok = true
			break
		}
	}
	return
}
