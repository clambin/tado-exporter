package statemanager

import (
	"github.com/clambin/tado-exporter/controller/models"
	"github.com/clambin/tado-exporter/controller/poller"
	"time"
)

func (mgr *Manager) GetNextState(zoneID int, update poller.Update) (nextState models.ZoneState, when time.Duration, reason string) {
	// if we don't trigger any rules, keep the same state
	nextState = update.ZoneStates[zoneID]

	// if all users are away -> set 'off'
	if mgr.zoneRules[zoneID].AutoAway.Enabled {
		if mgr.allUsersAway(zoneID, update) {
			nextState.State = models.ZoneOff
			when = mgr.zoneRules[zoneID].AutoAway.Delay
			reason = mgr.getAutoAwayReason(zoneID, false)
			return
		} else if update.ZoneStates[zoneID].State == models.ZoneOff {
			nextState.State = models.ZoneAuto
			// when = mgr.zoneRules[zoneID].AutoAway.Delay
			reason = mgr.getAutoAwayReason(zoneID, true)
		}
	}

	if update.ZoneStates[zoneID].State == models.ZoneManual {
		// determine if/when to set back to auto
		if mgr.zoneRules[zoneID].NightTime.Enabled && mgr.zoneRules[zoneID].LimitOverlay.Enabled {
			nextState.State = models.ZoneAuto
			when = nightTimeDelay(mgr.zoneRules[zoneID].NightTime.Time)
			if mgr.zoneRules[zoneID].LimitOverlay.Delay < when {
				when = mgr.zoneRules[zoneID].LimitOverlay.Delay
			}
			reason = "manual temperature setting detected in " + mgr.zoneNameCache[zoneID]
		} else if mgr.zoneRules[zoneID].NightTime.Enabled {
			nextState.State = models.ZoneAuto
			when = nightTimeDelay(mgr.zoneRules[zoneID].NightTime.Time)
			reason = "manual temperature setting detected in " + mgr.zoneNameCache[zoneID]
		} else if mgr.zoneRules[zoneID].LimitOverlay.Enabled {
			nextState.State = models.ZoneAuto
			when = mgr.zoneRules[zoneID].LimitOverlay.Delay
			reason = "manual temperature setting detected in " + mgr.zoneNameCache[zoneID]
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
	for _, user := range mgr.zoneRules[zoneID].AutoAway.Users {
		if update.UserStates[user] != models.UserAway {
			away = false
			break
		}
	}
	return
}

func (mgr *Manager) getAutoAwayReason(zoneID int, home bool) (reason string) {
	if len(mgr.zoneRules[zoneID].AutoAway.Users) == 1 {
		reason = mgr.userNameCache[mgr.zoneRules[zoneID].AutoAway.Users[0]] + " is "
		if home {
			reason += "home"
		} else {
			reason += "away"
		}
	} else {
		if home {
			reason = "one or more users are home"
		} else {
			reason = "all users of are away"
		}
	}
	reason = mgr.zoneNameCache[zoneID] + ": " + reason
	return
}
