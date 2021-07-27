package zonemanager

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/namecache"
	"github.com/clambin/tado-exporter/controller/scheduler"
	"github.com/clambin/tado-exporter/controller/statemanager"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	tado.API
	Update       chan *poller.Update
	Report       chan struct{}
	PostChannel  slackbot.PostChannel
	scheduler    *scheduler.Scheduler
	stateManager *statemanager.Manager
	cache        *namecache.Cache
}

func New(API tado.API, zoneConfig []configuration.ZoneConfig, postChannel slackbot.PostChannel) (mgr *Manager, err error) {
	cache := namecache.New()
	var stateManager *statemanager.Manager
	stateManager, err = statemanager.New(zoneConfig, cache)

	if err == nil {
		mgr = &Manager{
			API:          API,
			Update:       make(chan *poller.Update),
			Report:       make(chan struct{}),
			scheduler:    scheduler.New(),
			stateManager: stateManager,
			PostChannel:  postChannel,
			cache:        cache,
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
			mgr.cache.Update(update)
			mgr.update(ctx, update)
		case <-mgr.Report:
			mgr.reportTasks()
		}
	}
	log.Info("zone manager stopped")

	return
}

func (mgr *Manager) update(ctx context.Context, update *poller.Update) {
	for zoneID, zoneInfo := range update.ZoneInfo {

		state := zoneInfo.GetState()
		targetState, when, reason, err := mgr.stateManager.GetNextState(zoneID, update)

		if err != nil {
			log.WithError(err).Warning("failed to get zone state")
			continue
		}

		log.WithFields(log.Fields{"old": state, "new": targetState, "id": zoneID}).Debug("new zone state determined")

		if targetState != state {
			// schedule the new state
			mgr.scheduleZoneStateChange(ctx, zoneID, targetState, when, reason)
		} else {
			// already at target state. cancel any outstanding tasks
			mgr.cancelZoneStateChange(zoneID, reason)
		}
	}
	return
}
