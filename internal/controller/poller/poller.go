package poller

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
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
	ZoneStates map[int]models.ZoneState
	UserStates map[int]models.UserState
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
		zoneStates map[int]models.ZoneState
		userStates map[int]models.UserState
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

func (poller *Poller) getZoneStates() (states map[int]models.ZoneState, err error) {
	var zones []tado.Zone
	if zones, err = poller.API.GetZones(); err == nil {
		states = make(map[int]models.ZoneState)
		for _, zone := range zones {
			var state models.ZoneState
			if state, err = poller.getZoneState(zone.ID); err == nil {
				states[zone.ID] = state
			}
		}
	}
	return
}

func (poller *Poller) getZoneState(zoneID int) (state models.ZoneState, err error) {
	var zoneInfo tado.ZoneInfo
	if zoneInfo, err = poller.API.GetZoneInfo(zoneID); err == nil {
		state = models.GetZoneState(zoneInfo)
	}
	return
}

func (poller *Poller) getUserStates() (users map[int]models.UserState, err error) {
	var devices []tado.MobileDevice
	if devices, err = poller.API.GetMobileDevices(); err == nil {
		users = make(map[int]models.UserState)
		for _, device := range devices {
			state := models.UserUnknown
			if device.Settings.GeoTrackingEnabled == true {
				if device.Location.AtHome == false {
					state = models.UserAway
				} else {
					state = models.UserHome
				}
			}
			users[device.ID] = state
		}
	}
	return
}
