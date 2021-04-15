package poller

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
)

type Poller struct {
	API tado.API
}

type Update struct {
	ZoneStates map[int]models.ZoneState
	UserStates map[int]models.UserState
}

func New(API tado.API) *Poller {
	return &Poller{
		API: API,
	}
}

func (poller *Poller) Update() (update Update, err error) {
	var (
		zoneStates map[int]models.ZoneState
		userStates map[int]models.UserState
	)
	if zoneStates, err = poller.getZoneStates(); err == nil {
		if userStates, err = poller.getUserStates(); err == nil {
			update = Update{
				ZoneStates: zoneStates,
				UserStates: userStates,
			}
		}
	}

	if err != nil {
		log.WithField("err", err).Warning("failed to get tado status information")
	}
	return
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
			if device.Settings.GeoTrackingEnabled {
				var state models.UserState
				if device.Location.AtHome == false {
					state = models.UserAway
				} else {
					state = models.UserHome
				}
				users[device.ID] = state
			}
		}
	}
	return
}
