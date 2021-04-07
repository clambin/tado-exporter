package zonemanager

import "github.com/clambin/tado-exporter/internal/controller/tadoproxy"

func buildZoneLookup(proxy *tadoproxy.Proxy) map[int]string {
	response := make(chan map[int]string)
	proxy.GetAllZones <- response
	return <-response
}

func buildUserLookup(proxy *tadoproxy.Proxy) map[int]string {
	response := make(chan map[int]string)
	proxy.GetAllUsers <- response
	return <-response
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
