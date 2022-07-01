package setter

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// ZoneSetter receives the next zone state from Controller and sets the state at the appropriate time
type ZoneSetter interface {
	Set(nextState NextState)
	Clear(zoneID int)
	Run(ctx context.Context, interval time.Duration)
	GetScheduled() (scheduled map[int]NextState)
}

var _ ZoneSetter = &Server{}

// NextState describes the next State of a zone after a specified Delay
type NextState struct {
	ZoneID       int
	ZoneName     string
	State        tado.ZoneState
	Delay        time.Duration
	ActionReason string
	CancelReason string
	When         time.Time
}

// Server performs zone State changes received from Controller
type Server struct {
	tado.API
	slackbot.SlackBot
	tasks map[int]NextState
	lock  sync.RWMutex
}

// New creates a new setter Server
func New(API tado.API, bot slackbot.SlackBot) *Server {
	return &Server{
		API:      API,
		SlackBot: bot,
		tasks:    make(map[int]NextState),
	}
}

// Set implements the ZoneSetter interface. It registers the future state of the specified zone
func (server *Server) Set(nextState NextState) {
	server.lock.Lock()
	defer server.lock.Unlock()

	nextState.When = time.Now().Add(nextState.Delay)
	if current, ok := server.tasks[nextState.ZoneID]; ok {
		if current.State == nextState.State && current.When.Before(nextState.When) {
			// log.WithFields(log.Fields{"current": current, "task": nextState}).Info("earlier task exists. dropping")
			return
		}
	}
	log.WithFields(log.Fields{"zoneID": nextState.ZoneID, "state": nextState}).Info("queuing next state")
	server.tasks[nextState.ZoneID] = nextState
	if nextState.Delay > 0 {
		server.postAction(nextState)
	}
}

// Clear implements the ZoneSetter interface. It clears any future state of the specified zone
func (server *Server) Clear(zoneID int) {
	server.lock.Lock()
	defer server.lock.Unlock()
	if task, ok := server.tasks[zoneID]; ok {
		log.WithFields(log.Fields{"zoneID": zoneID, "task": task}).Info("removing queued next state")
		server.postCancel(task)
		delete(server.tasks, zoneID)
	}
}

// Run the server
func (server *Server) Run(ctx context.Context, interval time.Duration) {
	log.Info("setter started")
	ticker := time.NewTicker(interval)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-ticker.C:
			server.process(ctx)
		}
	}

	ticker.Stop()
	log.Info("setter stopped")
}

func (server *Server) GetScheduled() (scheduled map[int]NextState) {
	server.lock.RLock()
	defer server.lock.RUnlock()

	scheduled = make(map[int]NextState)
	for zoneID, nextState := range server.tasks {
		scheduled[zoneID] = nextState
	}

	return
}

func (server *Server) process(ctx context.Context) {
	server.lock.RLock()
	defer server.lock.RUnlock()

	var err error
	for zoneID, scheduledTask := range server.tasks {
		if time.Now().After(scheduledTask.When) {
			log.WithFields(log.Fields{"zoneID": zoneID, "state": scheduledTask}).Info("setting next state")
			switch scheduledTask.State {
			case tado.ZoneStateAuto:
				err = server.API.DeleteZoneOverlay(ctx, zoneID)
			case tado.ZoneStateOff:
				err = server.API.SetZoneOverlay(ctx, zoneID, 5.0)
			default:
				log.WithField("state", scheduledTask.State).Error("state not implemented")
			}
			if err == nil {
				scheduledTask.Delay = 0
				server.postAction(scheduledTask)
				delete(server.tasks, zoneID)
			} else {
				log.WithError(err).Warning("failed to call Tado")
			}
		}
	}
}

func (server *Server) postAction(nextState NextState) {
	var text string
	switch nextState.State {
	case tado.ZoneStateAuto:
		text = "moving to auto mode"
	case tado.ZoneStateOff:
		text = "switching off heating"
	}

	if nextState.Delay > 0 {
		text += " in " + nextState.Delay.Round(time.Second).String()
	}

	err := server.SlackBot.Send("", "good", nextState.ZoneName+": "+nextState.ActionReason, text)

	if err != nil {
		log.WithError(err).Warning("failed to post to slack")
	}
}

func (server *Server) postCancel(previousNextState NextState) {
	var text string
	switch previousNextState.State {
	case tado.ZoneStateAuto:
		text = "canceling task to move to auto mode"
	case tado.ZoneStateOff:
		text = "canceling task to switch off heating"
	}

	err := server.SlackBot.Send("", "good", previousNextState.ZoneName+": "+previousNextState.CancelReason, text)

	if err != nil {
		log.WithError(err).Warning("failed to post to slack")
	}
}
