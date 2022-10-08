package zonemanager

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"time"
)

// NextState describes the next State of a zone after a specified Delay
type NextState struct {
	ZoneID       int
	ZoneName     string
	State        tado.ZoneState
	Delay        time.Duration
	ActionReason string
	CancelReason string
}

func (m *Manager) getNextState(update *poller.Update) (current tado.ZoneState, nextState NextState) {
	current = update.ZoneInfo[m.config.ZoneID].GetState()

	// if we don't trigger any rules, keep the same state
	nextState = NextState{
		ZoneID:   m.config.ZoneID,
		ZoneName: m.config.ZoneName,
		State:    current,
	}

	if m.config.AutoAway.Enabled {
		// if all users are away -> set 'off'
		if m.allUsersAway(update) {
			nextState.State = tado.ZoneStateOff
			nextState.Delay = m.config.AutoAway.Delay
			nextState.ActionReason, nextState.CancelReason = m.getAutoAwayReason(false)

			return
		} else
		// if at least one user is home, set to 'auto'
		if nextState.State == tado.ZoneStateOff {
			nextState.State = tado.ZoneStateAuto
			nextState.ActionReason, nextState.CancelReason = m.getAutoAwayReason(true)
		}
	}

	if nextState.State != tado.ZoneStateManual {
		return
	}

	// determine if/when to set back to auto
	if m.config.NightTime.Enabled || m.config.LimitOverlay.Enabled {
		nextState.State = tado.ZoneStateAuto
		nextState.ActionReason = "manual temperature setting detected"
		nextState.CancelReason = "room is now in auto mode"

		var nightDelay, limitDelay time.Duration
		if m.config.NightTime.Enabled {
			nightDelay = nightTimeDelay(m.config.NightTime.Time, time.Now())
			nextState.Delay = nightDelay
		}
		if m.config.LimitOverlay.Enabled {
			limitDelay = m.config.LimitOverlay.Delay
			nextState.Delay = limitDelay
		}

		if nightDelay != 0 && limitDelay != 0 && nightDelay < nextState.Delay {
			nextState.Delay = nightDelay

		}
	}

	return
}

func nightTimeDelay(nightTime configuration.ZoneNightTimeTimestamp, now time.Time) (delay time.Duration) {
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		nightTime.Hour, nightTime.Minutes, 0, 0, time.Local)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (m *Manager) allUsersAway(update *poller.Update) bool {
	for _, user := range m.config.AutoAway.Users {
		device := update.UserInfo[user.MobileDeviceID]
		if device.IsHome() != tado.DeviceAway {
			return false
		}
	}
	return true
}

func (m *Manager) getAutoAwayReason(home bool) (actionReason, cancelReason string) {
	if len(m.config.AutoAway.Users) == 1 {
		actionReason = m.config.AutoAway.Users[0].MobileDeviceName + " is "
		cancelReason = actionReason
		if home {
			actionReason += "home"
			cancelReason += "away"
		} else {
			actionReason += "away"
			cancelReason += "home"

		}
	} else {
		if home {
			actionReason = "one or more users are home"
			cancelReason = "all users are away"
		} else {
			actionReason = "all users are away"
			cancelReason = "one or more users are home"
		}
	}
	return
}
