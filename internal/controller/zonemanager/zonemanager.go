package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/tadoproxy"
	log "github.com/sirupsen/logrus"
	"time"
)

type Manager struct {
	ZoneConfig map[int]model.ZoneConfig

	proxy     *tadoproxy.Proxy
	expiry    map[int]time.Time
	nightTime map[int]time.Time
	AllZones  map[int]string
	AllUsers  map[int]string
}

func New(zoneConfig []configuration.ZoneConfig, proxy *tadoproxy.Proxy) *Manager {
	mgr := &Manager{
		proxy:     proxy,
		expiry:    make(map[int]time.Time),
		nightTime: make(map[int]time.Time),
		AllZones:  proxy.GetAllZones(),
		AllUsers:  proxy.GetAllUsers(),
	}
	mgr.ZoneConfig = makeZoneConfig(zoneConfig, mgr.AllZones, mgr.AllUsers)

	return mgr
}

func (mgr *Manager) Update() (changes map[int]model.ZoneState) {
	changes = make(map[int]model.ZoneState)

	response := make(chan map[int]model.ZoneState)
	mgr.proxy.GetZones <- response

	for zoneID, state := range <-response {
		newState := mgr.newZoneState(zoneID, state)

		if newState != state {
			changes[zoneID] = newState
		}
	}
	return
}

func (mgr *Manager) newZoneState(zoneID int, state model.ZoneState) (newState model.ZoneState) {
	// if we don't trigger any rules, keep the same state
	newState = state

	// if all users are away -> set 'off'
	if mgr.allUsersAway(zoneID) {
		newState.State = model.Off
	} else {
		switch state.State {
		case model.Off:
			// one or more users are now home. switch back to auto
			newState.State = model.Auto
		case model.Manual:
			// in manual and after next nighttime. set to auto
			if mgr.isNightTime(zoneID) {
				newState.State = model.Auto
			}
			// if manual & longer than max time -> set to auto
			if mgr.isZoneOverlayExpired(zoneID) {
				newState.State = model.Auto
			}
		case model.Auto:
			// zone is in auto mode, so remove any timers
			delete(mgr.expiry, zoneID)
			delete(mgr.nightTime, zoneID)
		}
	}
	return
}

func (mgr *Manager) allUsersAway(zoneID int) (away bool) {
	if config, ok := mgr.ZoneConfig[zoneID]; ok == true && len(config.Users) > 0 {

		responseChannel := make(chan map[int]model.UserState)
		mgr.proxy.GetUsers <- responseChannel
		userStates := <-responseChannel

		away = true

		for _, userID := range config.Users {
			if userStates[userID] != model.UserAway {
				away = false
				break
			}
		}
	}
	return
}

func (mgr *Manager) isZoneOverlayExpired(zoneID int) (expired bool) {
	if config, configured := mgr.ZoneConfig[zoneID]; configured == true && config.LimitOverlay.Enabled == true {
		if expiry, ok := mgr.expiry[zoneID]; ok == false {
			mgr.expiry[zoneID] = time.Now().Add(config.LimitOverlay.Limit)

			log.WithFields(log.Fields{
				"zone":  mgr.AllZones[zoneID],
				"timer": mgr.expiry[zoneID],
			}).Info("setting expiry timer for zone in overlay")
		} else {
			now := time.Now()
			if now.After(expiry) {
				delete(mgr.expiry, zoneID)
				expired = true

				log.WithField("zone", mgr.AllZones[zoneID]).Info("timer expired for zone in overlay")
			}
		}
	}
	return
}

func (mgr *Manager) isNightTime(zoneID int) (nightMode bool) {
	if config, configured := mgr.ZoneConfig[zoneID]; configured && config.NightTime.Enabled {
		if nightTime, ok := mgr.nightTime[zoneID]; ok == false {
			now := time.Now()
			nightTime = time.Date(
				now.Year(), now.Month(), now.Day(),
				config.NightTime.Time.Hour, config.NightTime.Time.Minutes, 0, 0, time.Local)
			if now.After(nightTime) {
				nightTime.Add(24 * time.Hour)
			}
			mgr.nightTime[zoneID] = nightTime
		} else if time.Now().After(nightTime) {
			delete(mgr.nightTime, zoneID)
			nightMode = true
		}
	}
	return
}
