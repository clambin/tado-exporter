package poller

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

type Poller struct {
	API    tado.API
	Cancel chan struct{}
	Update chan Update
	ticker *time.Ticker
}

type Update struct {
	ZoneStates map[int]model.ZoneState
	UserStates map[int]model.UserState
}

func New(API tado.API, interval time.Duration) *Poller {
	return &Poller{
		API:    API,
		Update: make(chan Update),
		Cancel: make(chan struct{}),
		ticker: time.NewTicker(interval),
	}
}

func (poller *Poller) Run() {
loop:
	for {
		select {
		case <-poller.ticker.C:
			poller.update()
		case <-poller.Cancel:
			break loop
		}
	}
	poller.ticker.Stop()
	close(poller.Cancel)
}

func (poller *Poller) update() {
	var (
		err        error
		zoneStates map[int]model.ZoneState
		userStates map[int]model.UserState
	)
	if zoneStates, err = poller.getZoneStates(); err == nil {
		if userStates, err = poller.getUserStates(); err == nil {
			poller.Update <- Update{
				ZoneStates: zoneStates,
				UserStates: userStates,
			}
		}
	}

	if err != nil {
		log.WithField("err", err).Warning("failed to get tado status information")
	}
}

func (poller *Poller) getZoneStates() (states map[int]model.ZoneState, err error) {
	var zones []tado.Zone
	if zones, err = poller.API.GetZones(); err == nil {
		states = make(map[int]model.ZoneState)
		for _, zone := range zones {
			var state model.ZoneState
			if state, err = poller.getZoneState(zone.ID); err == nil {
				states[zone.ID] = state
			}
		}
	}
	return
}

func (poller *Poller) getZoneState(zoneID int) (state model.ZoneState, err error) {
	var zoneInfo tado.ZoneInfo
	if zoneInfo, err = poller.API.GetZoneInfo(zoneID); err == nil {
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
	}
	return
}

func (poller *Poller) getUserStates() (users map[int]model.UserState, err error) {
	var devices []tado.MobileDevice
	if devices, err = poller.API.GetMobileDevices(); err == nil {
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
	}
	return
}
