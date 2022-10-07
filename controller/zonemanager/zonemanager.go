package zonemanager

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Manager struct {
	Updates   chan *poller.Update
	config    configuration.ZoneConfig
	scheduler scheduler.Scheduler
	job       *Job
	api       tado.API
	poster    Poster
	poller    poller.Poller
	loaded    bool
	lock      sync.RWMutex
}

func New(api tado.API, p poller.Poller, bot slackbot.SlackBot, cfg configuration.ZoneConfig) *Manager {
	return &Manager{
		Updates: make(chan *poller.Update, 1),
		config:  cfg,
		api:     api,
		poster:  Poster{SlackBot: bot},
		poller:  p,
	}
}

func (m *Manager) Run(ctx context.Context, interval time.Duration) {
	m.poller.Register(m.Updates)

	ticker := time.NewTicker(interval)
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case update := <-m.Updates:
			if err := m.process(ctx, update); err != nil {
				log.WithError(err).WithField("zone", m.config.ZoneName).Error("failed to process tado update")
			}
		case <-ticker.C:
			if completed, err := m.scheduler.Result(); completed {
				if err == nil {
					m.poster.NotifyAction(m.job.nextState)
				} else {
					log.WithError(err).WithField("zone", m.config.ZoneName).Error("failed to set next state")
					// TODO: reschedule the failed job?
				}
				m.job = nil
			}
		}
	}
	ticker.Stop()

	m.poller.Unregister(m.Updates)
}

func (m *Manager) process(ctx context.Context, update *poller.Update) (err error) {
	if err = m.load(update); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	// next state
	current, next := m.getNextState(update)

	if current != next.State {
		m.scheduleJob(ctx, next)
	} else {
		m.cancelJob()
	}

	return
}

func (m *Manager) scheduleJob(ctx context.Context, next NextState) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// if the same state is already scheduled for an earlier time, don't schedule it again.
	if m.job != nil && m.job.nextState.State == next.State && next.Delay >= time.Until(m.job.when) {
		return
	}

	m.job = &Job{
		api:       m.api,
		nextState: next,
		when:      time.Now().Add(next.Delay),
	}
	m.scheduler.Schedule(ctx, m.job.Run, next.Delay)
	if next.Delay > 0 {
		m.poster.NotifyQueued(next)
	}
}

func (m *Manager) cancelJob() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.scheduler.Cancel()
	if m.job != nil {
		m.poster.NotifyCanceled(m.job.nextState)
		m.job = nil
	}
}

func (m *Manager) Scheduled() (NextState, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.job != nil {
		return m.job.nextState, true
	}
	return NextState{}, false
}

func (m *Manager) ReportTask() (string, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.job == nil {
		return "", false
	}

	var action string
	switch m.job.nextState.State {
	case tado.ZoneStateOff:
		action = "switching off heating"
	case tado.ZoneStateAuto:
		action = "moving to auto mode"
	}

	return m.job.nextState.ZoneName + ": " + action + " in " + time.Until(m.job.when).Round(time.Second).String(), true
}

type Job struct {
	api       tado.API
	nextState NextState
	when      time.Time
}

func (j *Job) Run(ctx context.Context) (err error) {
	switch j.nextState.State {
	case tado.ZoneStateAuto:
		err = j.api.DeleteZoneOverlay(ctx, j.nextState.ZoneID)
	case tado.ZoneStateOff:
		err = j.api.SetZoneOverlay(ctx, j.nextState.ZoneID, 5.0)
	default:
		err = fmt.Errorf("invalid queued state for zone '%s': %d", j.nextState.ZoneName, j.nextState.State)
	}
	return
}

type Managers []*Manager

func (m Managers) GetScheduled() (states []NextState) {
	for _, mgr := range m {
		if state, scheduled := mgr.Scheduled(); scheduled {
			states = append(states, state)
		}
	}
	return
}

func (m Managers) ReportTasks() ([]string, bool) {
	var tasks []string
	for _, mgr := range m {
		if task, ok := mgr.ReportTask(); ok {
			tasks = append(tasks, task)
		}
	}
	return tasks, len(tasks) > 0
}
