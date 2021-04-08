package tadoproxy

import "github.com/clambin/tado-exporter/internal/controller/model"

func (proxy *Proxy) GetAllZones() map[int]string {
	response := make(chan map[int]string)
	proxy.AllZones <- response
	return <-response
}

func (proxy *Proxy) GetAllUsers() map[int]string {
	response := make(chan map[int]string)
	proxy.AllUsers <- response
	return <-response
}

func (proxy *Proxy) GetAllZoneStates() map[int]model.ZoneState {
	response := make(chan map[int]model.ZoneState)
	proxy.GetZones <- response
	return <-response
}

func (proxy *Proxy) GetAllUserStates() map[int]model.UserState {
	response := make(chan map[int]model.UserState)
	proxy.GetUsers <- response
	return <-response
}
