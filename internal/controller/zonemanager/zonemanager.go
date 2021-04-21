package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/models"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"time"
)

type Manager struct {
	API         tado.API
	ZoneConfig  map[int]models.ZoneConfig
	Cancel      chan struct{}
	Update      chan poller.Update
	Report      chan struct{}
	fire        chan *Task
	tasks       map[int]*Task
	nameCache   map[int]string
	postChannel slackbot.PostChannel
}

func New(API tado.API, zoneConfig []configuration.ZoneConfig, postChannel slackbot.PostChannel) (mgr *Manager, err error) {
	mgr = &Manager{
		API:    API,
		Cancel: make(chan struct{}),
		Update: make(chan poller.Update),
		Report: make(chan struct{}),

		fire:        make(chan *Task),
		tasks:       make(map[int]*Task),
		postChannel: postChannel,
		nameCache:   make(map[int]string),
	}
	mgr.ZoneConfig, err = mgr.makeZoneConfig(zoneConfig)

	return
}

func (mgr *Manager) Run() {
loop:
	for {
		select {
		case <-mgr.Cancel:
			break loop
		case update := <-mgr.Update:
			mgr.update(update)
		case task := <-mgr.fire:
			mgr.runTask(task)
		case <-mgr.Report:
			mgr.reportTasks()
		}
	}
}

func (mgr *Manager) update(update poller.Update) {
	for zoneID, state := range update.ZoneStates {
		if _, ok := mgr.ZoneConfig[zoneID]; ok == true {
			targetState, when, reason := mgr.newZoneState(zoneID, update)
			if targetState.Equals(state) == false {
				// schedule the new state
				mgr.scheduleTask(zoneID, targetState, when, reason)
			} else {
				// already at target state. cancel any outstanding tasks
				mgr.unscheduleTask(zoneID, reason)
			}
		}
	}
	return
}

func (mgr *Manager) newZoneState(zoneID int, update poller.Update) (newState models.ZoneState, when time.Duration, reason string) {
	// if we don't trigger any rules, keep the same state
	newState = update.ZoneStates[zoneID]

	// if all users are away -> set 'off'
	if mgr.ZoneConfig[zoneID].AutoAway.Enabled {
		if mgr.allUsersAway(zoneID, update) {
			newState.State = models.ZoneOff
			when = mgr.ZoneConfig[zoneID].AutoAway.Delay
			reason = "all users of " + mgr.getZoneName(zoneID) + " are away"
			return
		} else if update.ZoneStates[zoneID].State == models.ZoneOff {
			newState.State = models.ZoneAuto
			when = mgr.ZoneConfig[zoneID].AutoAway.Delay
			reason = "one or more users of " + mgr.getZoneName(zoneID) + " are home"
		}
	}

	if update.ZoneStates[zoneID].State == models.ZoneManual {
		// determine if/when to set back to auto
		if mgr.ZoneConfig[zoneID].NightTime.Enabled && mgr.ZoneConfig[zoneID].LimitOverlay.Enabled {
			newState.State = models.ZoneAuto
			when = nightTimeDelay(mgr.ZoneConfig[zoneID].NightTime.Time)
			if mgr.ZoneConfig[zoneID].LimitOverlay.Delay < when {
				when = mgr.ZoneConfig[zoneID].LimitOverlay.Delay
			}
			reason = "manual temperature setting detected in " + mgr.getZoneName(zoneID)
		} else if mgr.ZoneConfig[zoneID].NightTime.Enabled {
			newState.State = models.ZoneAuto
			when = nightTimeDelay(mgr.ZoneConfig[zoneID].NightTime.Time)
			reason = "manual temperature setting detected in " + mgr.getZoneName(zoneID)
		} else if mgr.ZoneConfig[zoneID].LimitOverlay.Enabled {
			newState.State = models.ZoneAuto
			when = mgr.ZoneConfig[zoneID].LimitOverlay.Delay
			reason = "manual temperature setting detected in " + mgr.getZoneName(zoneID)
		}
	}
	return
}

func nightTimeDelay(nightTime models.ZoneNightTimestamp) (delay time.Duration) {
	now := time.Now()
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		nightTime.Hour, nightTime.Minutes, 0, 0, time.Local)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (mgr *Manager) allUsersAway(zoneID int, update poller.Update) (away bool) {
	away = true
	for _, user := range mgr.ZoneConfig[zoneID].AutoAway.Users {
		if update.UserStates[user] != models.UserAway {
			away = false
			break
		}
	}
	return
}

func (mgr *Manager) getZoneName(zoneID int) (name string) {
	var ok bool
	if name, ok = mgr.nameCache[zoneID]; ok == false {
		name = "unknown"
		if zones, err := mgr.API.GetZones(); err == nil {
			for _, zone := range zones {
				if zone.ID == zoneID {
					name = zone.Name
					mgr.nameCache[zoneID] = name
					break
				}
			}
		}
	}
	return
}
