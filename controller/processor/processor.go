package processor

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/setter"
	"github.com/clambin/tado-exporter/poller"
	log "github.com/sirupsen/logrus"
	"time"
)

// Processor receives the latest zone status and determines the next zone state
type Processor interface {
	Process(update *poller.Update) (nextStates map[int]*setter.NextState)
}

var _ Processor = &Server{}

// Server receives the latest zone status and determines the next zone state
type Server struct {
	zoneConfig []configuration.ZoneConfig
	zoneRules  map[int]ZoneRules
}

// New creates a new Server
func New(zoneConfig []configuration.ZoneConfig) (server *Server) {
	return &Server{zoneConfig: zoneConfig}
}

// Process processes a poller's Update and sets the next state for each zone
func (server *Server) Process(update *poller.Update) (nextStates map[int]*setter.NextState) {
	if server.zoneRules == nil {
		server.load(update)
	}

	nextStates = make(map[int]*setter.NextState)

	for zoneID := range update.ZoneInfo {
		current, nextState, err := server.getNextState(zoneID, update)

		if err != nil {
			log.WithError(err).WithField("zoneID", zoneID).Warning("failed to determine zone's next state")
			continue
		}

		log.WithFields(log.Fields{"ZoneID": zoneID, "current": current, "next": nextState.State}).Debug("processing")

		if nextState.State != current {
			nextStates[zoneID] = &nextState
		} else {
			nextStates[zoneID] = nil
		}
	}
	return
}

func (server *Server) getNextState(zoneID int, update *poller.Update) (current tado.ZoneState, nextState setter.NextState, err error) {
	nextState.ZoneID = zoneID
	nextState.ZoneName = update.Zones[zoneID].Name

	// if we don't trigger any rules, keep the same state
	zoneInfo := update.ZoneInfo[zoneID]
	current = zoneInfo.GetState()
	nextState.State = current

	// if all users are away -> set 'off'
	if server.zoneRules[zoneID].AutoAway.Enabled {
		if server.allUsersAway(zoneID, update) {
			nextState.State = tado.ZoneStateOff
			nextState.Delay = server.zoneRules[zoneID].AutoAway.Delay
			nextState.ActionReason, nextState.CancelReason = server.getAutoAwayReason(zoneID, false, update)

			return
		} else if nextState.State == tado.ZoneStateOff {
			nextState.State = tado.ZoneStateAuto
			nextState.ActionReason, nextState.CancelReason = server.getAutoAwayReason(zoneID, true, update)
		}
	}

	if nextState.State == tado.ZoneStateManual {
		// determine if/when to set back to auto
		if server.zoneRules[zoneID].NightTime.Enabled || server.zoneRules[zoneID].LimitOverlay.Enabled {
			nextState.State = tado.ZoneStateAuto
			nextState.ActionReason = "manual temperature setting detected"
			nextState.CancelReason = "room is now in auto mode"

			var nightDelay, limitDelay time.Duration
			if server.zoneRules[zoneID].NightTime.Enabled {
				nightDelay = nightTimeDelay(server.zoneRules[zoneID].NightTime.Time)
				nextState.Delay = nightDelay
			}
			if server.zoneRules[zoneID].LimitOverlay.Enabled {
				limitDelay = server.zoneRules[zoneID].LimitOverlay.Delay
				nextState.Delay = limitDelay
			}

			if server.zoneRules[zoneID].NightTime.Enabled && server.zoneRules[zoneID].LimitOverlay.Enabled {
				if nightDelay < nextState.Delay {
					nextState.Delay = nightDelay
				}
			}
		}
	}
	return
}

func nightTimeDelay(nightTime ZoneNightTimestamp) (delay time.Duration) {
	now := time.Now()
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		nightTime.Hour, nightTime.Minutes, 0, 0, time.Local)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (server *Server) allUsersAway(zoneID int, update *poller.Update) (away bool) {
	away = true
	for _, user := range server.zoneRules[zoneID].AutoAway.Users {
		device := update.UserInfo[user]
		if (&device).IsHome() != tado.DeviceAway {
			away = false
			break
		}
	}
	return
}

func (server *Server) getAutoAwayReason(zoneID int, home bool, update *poller.Update) (actionReason, cancelReason string) {
	if len(server.zoneRules[zoneID].AutoAway.Users) == 1 {
		actionReason = update.UserInfo[server.zoneRules[zoneID].AutoAway.Users[0]].Name + " is "
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
