package statemanager

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/models"
	"github.com/clambin/tado-exporter/poller"
	"time"
)

func (mgr *Manager) GetNextState(zoneID int, update *poller.Update) (nextState tado.ZoneState, when time.Duration, reason string, err error) {
	mgr.initialize()

	zoneInfo, ok := update.ZoneInfo[zoneID]

	if ok == false {
		err = fmt.Errorf("zone ID %d not found in Tado metrics", zoneID)
		return
	}

	// if we don't trigger any rules, keep the same state
	currentState := zoneInfo.GetState()
	nextState = currentState

	// if all users are away -> set 'off'
	if mgr.zoneRules[zoneID].AutoAway.Enabled {
		if mgr.allUsersAway(zoneID, update) {
			nextState = tado.ZoneStateOff
			when = mgr.zoneRules[zoneID].AutoAway.Delay
			reason = mgr.getAutoAwayReason(zoneID, false, update)
			return
		} else if currentState == tado.ZoneStateOff {
			nextState = tado.ZoneStateAuto
			// when = mgr.zoneRules[zoneID].AutoAway.Delay
			reason = mgr.getAutoAwayReason(zoneID, true, update)
		}
	}

	if currentState == tado.ZoneStateManual {
		// determine if/when to set back to auto
		if mgr.zoneRules[zoneID].NightTime.Enabled && mgr.zoneRules[zoneID].LimitOverlay.Enabled {
			nextState = tado.ZoneStateAuto
			when = nightTimeDelay(mgr.zoneRules[zoneID].NightTime.Time)
			if mgr.zoneRules[zoneID].LimitOverlay.Delay < when {
				when = mgr.zoneRules[zoneID].LimitOverlay.Delay
			}
			reason = "manual temperature setting detected in " + update.Zones[zoneID].Name
		} else if mgr.zoneRules[zoneID].NightTime.Enabled {
			nextState = tado.ZoneStateAuto
			when = nightTimeDelay(mgr.zoneRules[zoneID].NightTime.Time)
			reason = "manual temperature setting detected in " + update.Zones[zoneID].Name
		} else if mgr.zoneRules[zoneID].LimitOverlay.Enabled {
			nextState = tado.ZoneStateAuto
			when = mgr.zoneRules[zoneID].LimitOverlay.Delay
			reason = "manual temperature setting detected in " + update.Zones[zoneID].Name
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

func (mgr *Manager) allUsersAway(zoneID int, update *poller.Update) (away bool) {
	away = true
	for _, user := range mgr.zoneRules[zoneID].AutoAway.Users {
		device := update.UserInfo[user]
		if (&device).IsHome() != tado.DeviceAway {
			away = false
			break
		}
	}
	return
}

func (mgr *Manager) getAutoAwayReason(zoneID int, home bool, update *poller.Update) (reason string) {
	if len(mgr.zoneRules[zoneID].AutoAway.Users) == 1 {
		reason = update.UserInfo[mgr.zoneRules[zoneID].AutoAway.Users[0]].Name + " is "
		if home {
			reason += "home"
		} else {
			reason += "away"
		}
	} else {
		if home {
			reason = "one or more users are home"
		} else {
			reason = "all users are away"
		}
	}
	reason = update.Zones[zoneID].Name + ": " + reason
	return
}
