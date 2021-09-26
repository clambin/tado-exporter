package processor

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/models"
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
	zoneRules  map[int]models.ZoneRules
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
		current, next, delay, reason, err := server.getNextState(zoneID, update)

		if err != nil {
			log.WithError(err).WithField("zoneID", zoneID).Warning("failed to determine zone's next state")
			continue
		}

		// log.WithFields(log.Fields{"ZoneID": zoneID, "current": current, "next": next}).Debug("processing")

		if next != current {
			nextStates[zoneID] = &setter.NextState{State: next, Delay: delay, Reason: reason}
		} else {
			nextStates[zoneID] = nil
		}
	}
	return
}

func (server *Server) getNextState(zoneID int, update *poller.Update) (current, nextState tado.ZoneState, delay time.Duration, reason string, err error) {
	zoneInfo, _ := update.ZoneInfo[zoneID]

	// if we don't trigger any rules, keep the same state
	current = zoneInfo.GetState()
	nextState = current

	// if all users are away -> set 'off'
	if server.zoneRules[zoneID].AutoAway.Enabled {
		if server.allUsersAway(zoneID, update) {
			nextState = tado.ZoneStateOff
			delay = server.zoneRules[zoneID].AutoAway.Delay
			reason = server.getAutoAwayReason(zoneID, false, update)
			return
		} else if current == tado.ZoneStateOff {
			nextState = tado.ZoneStateAuto
			reason = server.getAutoAwayReason(zoneID, true, update)
		}
	}

	if current == tado.ZoneStateManual {
		log.WithField("zoneID", zoneID).Debug("zone in overlay")
		// determine if/when to set back to auto
		if server.zoneRules[zoneID].NightTime.Enabled || server.zoneRules[zoneID].LimitOverlay.Enabled {
			nextState = tado.ZoneStateAuto
			reason = "manual temperature setting detected in " + update.Zones[zoneID].Name

			var nightDelay, limitDelay time.Duration
			if server.zoneRules[zoneID].NightTime.Enabled {
				nightDelay = nightTimeDelay(server.zoneRules[zoneID].NightTime.Time)
				delay = nightDelay
			}
			if server.zoneRules[zoneID].LimitOverlay.Enabled {
				limitDelay = server.zoneRules[zoneID].LimitOverlay.Delay
				delay = limitDelay
			}

			if server.zoneRules[zoneID].NightTime.Enabled && server.zoneRules[zoneID].LimitOverlay.Enabled {
				if nightDelay < delay {
					delay = nightDelay
				}
			}
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

func (server *Server) getAutoAwayReason(zoneID int, home bool, update *poller.Update) (reason string) {
	if len(server.zoneRules[zoneID].AutoAway.Users) == 1 {
		reason = update.UserInfo[server.zoneRules[zoneID].AutoAway.Users[0]].Name + " is "
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
