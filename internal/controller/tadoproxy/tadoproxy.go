package tadoproxy

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type Proxy struct {
	tado.API

	GetZones    chan chan map[int]model.ZoneState
	SetZones    chan map[int]model.ZoneState
	GetUsers    chan chan map[int]model.UserState
	GetAllZones chan chan map[int]string
	GetAllUsers chan chan map[int]string

	Stop chan struct{}
}

func New(tadoUsername, tadoPassword, tadoClientSecret string) *Proxy {
	return &Proxy{
		API: &tado.APIClient{
			HTTPClient:   &http.Client{},
			Username:     tadoUsername,
			Password:     tadoPassword,
			ClientSecret: tadoClientSecret,
		},
		GetZones:    make(chan chan map[int]model.ZoneState),
		SetZones:    make(chan map[int]model.ZoneState),
		GetUsers:    make(chan chan map[int]model.UserState),
		GetAllZones: make(chan chan map[int]string),
		GetAllUsers: make(chan chan map[int]string),
		Stop:        make(chan struct{}),
	}
}

func (proxy *Proxy) Run() {
loop:
	for {
		select {
		case states := <-proxy.SetZones:
			proxy.setStates(states)
		case responseChannel := <-proxy.GetZones:
			responseChannel <- proxy.getZoneStates()
		case responseChannel := <-proxy.GetUsers:
			responseChannel <- proxy.getUserStates()
		case responseChannel := <-proxy.GetAllZones:
			responseChannel <- proxy.getAllZones()
		case responseChannel := <-proxy.GetAllUsers:
			responseChannel <- proxy.getAllUsers()
		case <-proxy.Stop:
			break loop
		}
	}

	close(proxy.GetZones)
	close(proxy.SetZones)
	close(proxy.Stop)
}

func (proxy *Proxy) setStates(states map[int]model.ZoneState) {
	for zoneID, state := range states {
		proxy.setZoneState(zoneID, state)
	}
}

func (proxy *Proxy) setZoneState(zoneID int, state model.ZoneState) {
	var err error
	switch state.State {
	case model.Off:
		err = proxy.SetZoneOverlay(zoneID, 5.0)
	case model.Auto:
		err = proxy.DeleteZoneOverlay(zoneID)
	case model.Manual:
		err = proxy.SetZoneOverlay(zoneID, state.Temperature.Celsius)
	}

	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"zoneID": zoneID,
			"state":  state.String(),
		}).Warning("failed to set zone state")
	}
}

func (proxy *Proxy) getZoneStates() (states map[int]model.ZoneState) {
	if zones, err := proxy.API.GetZones(); err == nil {
		states = make(map[int]model.ZoneState)
		for _, zone := range zones {
			if state, err := proxy.getZoneState(zone.ID); err == nil {
				states[zone.ID] = state
			}
		}
	} else {
		log.WithField("err", err).Warning("failed to get zones")
	}

	return
}

func (proxy *Proxy) getZoneState(zoneID int) (state model.ZoneState, err error) {
	var zoneInfo *tado.ZoneInfo

	if zoneInfo, err = proxy.API.GetZoneInfo(zoneID); err == nil {
		if zoneInfo.Overlay.Type == "MANUAL" &&
			zoneInfo.Overlay.Setting.Type == "HEATING" &&
			zoneInfo.Overlay.Termination.Type == "MANUAL" {
			if zoneInfo.Overlay.Setting.Temperature.Celsius == 5.0 {
				// TODO: probably more states that should be considered "off"?
				state.State = model.Off
			} else {
				state.State = model.Manual
				state.Temperature = zoneInfo.Overlay.Setting.Temperature
			}
		} else {
			state.State = model.Auto
		}
	} else {
		log.WithField("err", err).Warning("failed to get zone information")
	}

	return
}

func (proxy *Proxy) getUserStates() (users map[int]model.UserState) {
	if devices, err := proxy.API.GetMobileDevices(); err == nil {
		users = make(map[int]model.UserState)
		for _, device := range devices {
			state := model.UserUnknown

			if device.Settings.GeoTrackingEnabled == true {
				if device.Location.AtHome == false {
					state = model.UserAway
				} else {
					state = model.UserHome
				}
			}
			users[device.ID] = state
		}

	} else {
		log.WithField("err", err).Warning("failed to get mobile devices")
	}
	return
}

func (proxy *Proxy) getAllZones() (allZones map[int]string) {
	allZones = make(map[int]string)

	if zones, err := proxy.API.GetZones(); err == nil {
		for _, zone := range zones {
			allZones[zone.ID] = zone.Name
		}
	} else {
		log.WithField("err", err).Warning("failed to get all zones")
	}
	return
}

func (proxy *Proxy) getAllUsers() (allUsers map[int]string) {
	allUsers = make(map[int]string)

	if mobileDevices, err := proxy.API.GetMobileDevices(); err == nil {
		for _, mobileDevice := range mobileDevices {
			allUsers[mobileDevice.ID] = mobileDevice.Name
		}
	} else {
		log.WithField("err", err).Warning("failed to get all mobile devices")
	}
	return
}
