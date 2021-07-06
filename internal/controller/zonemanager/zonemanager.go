package zonemanager

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/poller"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/statemanager"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	API         tado.API
	Update      chan poller.Update
	Report      chan struct{}
	initialized bool
	// lock          sync.RWMutex
	scheduler    *scheduler.Scheduler
	stateManager *statemanager.Manager
	postChannel  slackbot.PostChannel
}

func New(API tado.API, zoneConfig []configuration.ZoneConfig, postChannel slackbot.PostChannel) (mgr *Manager, err error) {
	var stateManager *statemanager.Manager
	stateManager, err = statemanager.New(API, zoneConfig)

	if err == nil {
		mgr = &Manager{
			API:          API,
			Update:       make(chan poller.Update),
			Report:       make(chan struct{}),
			scheduler:    scheduler.New(),
			stateManager: stateManager,
			postChannel:  postChannel,
		}
	}
	return
}

func (mgr *Manager) Run(ctx context.Context) (err error) {
	log.Info("zone manager started")
	go mgr.scheduler.Run(ctx)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case update := <-mgr.Update:
			mgr.update(ctx, update)
		case <-mgr.Report:
			mgr.reportTasks()
		}
	}
	log.Info("zone manager stopped")

	return
}

func (mgr *Manager) update(ctx context.Context, update poller.Update) {
	for zoneID, state := range update.ZoneStates {
		if mgr.stateManager.IsValidZoneID(zoneID) == false {
			continue
		}

		targetState, when, reason := mgr.stateManager.GetNextState(zoneID, update)

		log.WithFields(log.Fields{"old": state, "new": targetState, "id": zoneID}).Debug("new zone state determined")

		if !targetState.Equals(state) {
			// schedule the new state
			mgr.scheduleZoneStateChange(ctx, zoneID, targetState, when, reason)
		} else {
			// already at target state. cancel any outstanding tasks
			mgr.cancelZoneStateChange(zoneID, reason)
		}
	}
	return
}
