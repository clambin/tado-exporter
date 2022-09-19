package zonemanager

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
	"time"
)

type Manager struct {
	Updates chan *poller.Update
	config  configuration.ZoneConfig
	queue   Queue
	poller  poller.Poller
	loaded  bool
}

func New(api tado.API, p poller.Poller, bot slackbot.SlackBot, cfg configuration.ZoneConfig) *Manager {
	return &Manager{
		Updates: make(chan *poller.Update, 1),
		config:  cfg,
		queue:   Queue{API: api, poster: Poster{SlackBot: bot}},
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
			if err := m.process(update); err != nil {
				log.WithError(err).WithField("zone", m.config.ZoneName).Error("failed to process tado update")
			}
		case <-ticker.C:
			if err := m.queue.Process(ctx); err != nil {
				log.WithError(err).WithField("zone", m.config.ZoneName).Error("failed to set next state")
			}
		}
	}
	ticker.Stop()

	m.poller.Unregister(m.Updates)
}

func (m *Manager) process(update *poller.Update) (err error) {
	if err = m.load(update); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}
	// next state
	current, next := m.getNextState(update)

	if current != next.State {
		m.queue.Queue(next)
	} else {
		m.queue.Clear()
	}

	return
}

func (m *Manager) Scheduled() (NextState, bool) {
	return m.queue.GetQueued()
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
