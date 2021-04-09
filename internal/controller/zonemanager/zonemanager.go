package zonemanager

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/pkg/tado"
	"time"
)

type Manager struct {
	API        tado.API
	ZoneConfig map[int]model.ZoneConfig
	Cancel     chan struct{}
	Update     chan poller.Update
	queued     map[int]model.ZoneState
	scheduler  chan *scheduler.Task
}

func New(API tado.API, zoneConfig []configuration.ZoneConfig, updater chan poller.Update, scheduler chan *scheduler.Task) (mgr *Manager, err error) {
	mgr = &Manager{
		API:       API,
		Cancel:    make(chan struct{}),
		Update:    updater,
		queued:    make(map[int]model.ZoneState),
		scheduler: scheduler,
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
		}
	}
}

func (mgr *Manager) update(update poller.Update) {
	for zoneID, state := range update.ZoneStates {
		newState, when := mgr.newZoneState(zoneID, update)

		if newState != state {
			// check if we've already queued this update with the scheduler
			if queuedState, ok := mgr.queued[zoneID]; !ok || queuedState != newState {
				// queue the update
				mgr.scheduler <- &scheduler.Task{
					ZoneID: zoneID,
					State:  newState,
					When:   when,
				}
				mgr.queued[zoneID] = newState
			}
		}

		// if we're back in auto mode, delete the queued state so we can switch back to overlay
		// would be cleaner to do this when the auto task has been processed by the scheduler
		if state.State == model.Auto && newState.State == model.Auto {
			delete(mgr.queued, zoneID)
		}
	}
	return
}

func (mgr *Manager) newZoneState(zoneID int, update poller.Update) (newState model.ZoneState, when time.Duration) {
	// if we don't trigger any rules, keep the same state
	newState = update.ZoneStates[zoneID]

	// if all users are away -> set 'off'
	if mgr.ZoneConfig[zoneID].AutoAway.Enabled {
		if mgr.allUsersAway(zoneID, update) {
			newState.State = model.Off
			when = mgr.ZoneConfig[zoneID].AutoAway.Delay
		} else {
			newState.State = model.Auto
		}
	}

	if update.ZoneStates[zoneID].State == model.Manual && newState.State != model.Off {
		// determine if/when to set back to auto
		if mgr.ZoneConfig[zoneID].NightTime.Enabled && mgr.ZoneConfig[zoneID].LimitOverlay.Enabled {
			newState.State = model.Auto
			when = nightTimeDelay(mgr.ZoneConfig[zoneID].NightTime.Time)
			if mgr.ZoneConfig[zoneID].LimitOverlay.Delay < when {
				when = mgr.ZoneConfig[zoneID].LimitOverlay.Delay
			}
		} else if mgr.ZoneConfig[zoneID].NightTime.Enabled {
			newState.State = model.Auto
			when = nightTimeDelay(mgr.ZoneConfig[zoneID].NightTime.Time)
		} else if mgr.ZoneConfig[zoneID].LimitOverlay.Enabled {
			newState.State = model.Auto
			when = mgr.ZoneConfig[zoneID].LimitOverlay.Delay
		}
	}
	return
}

func nightTimeDelay(nightTime model.ZoneNightTimestamp) (delay time.Duration) {
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
		if update.UserStates[user] == model.UserHome {
			away = false
			break
		}
	}
	return
}
