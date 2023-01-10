package zonemanager

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager/logger"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/clambin/tado-exporter/poller"
	"golang.org/x/exp/slog"
	"sync"
	"time"
)

type Manager struct {
	evaluator *rules.Evaluator
	task      *Task
	api       tado.API
	loggers   logger.Loggers
	poller    poller.Poller
	lock      sync.RWMutex
}

func New(api tado.API, p poller.Poller, bot slackbot.SlackBot, cfg configuration.ZoneConfig) *Manager {
	loggers := logger.Loggers{&logger.StdOutLogger{}}
	if bot != nil {
		loggers = append(loggers, &logger.SlackLogger{Bot: bot})
	}

	return &Manager{
		evaluator: &rules.Evaluator{Config: &cfg},
		api:       api,
		loggers:   loggers,
		poller:    p,
	}
}

func (m *Manager) Run(ctx context.Context, interval time.Duration) {
	ch := m.poller.Register()
	defer m.poller.Unregister(ch)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-ch:
			if err := m.processUpdate(ctx, update); err != nil {
				slog.Error("failed to process tado update", err, "zone", m.evaluator.ZoneName)
			}
		case <-ticker.C:
			if err := m.processResult(); err != nil {
				slog.Error("failed to set next state", err, "zone", m.evaluator.ZoneName)
			}
		}
	}
}

func (m *Manager) processUpdate(ctx context.Context, update *poller.Update) error {
	next, err := m.evaluator.Evaluate(update)
	if err != nil {
		return fmt.Errorf("failed to evaluate rules: %w", err)
	}

	if next != nil {
		m.scheduleJob(ctx, next)
	} else {
		m.cancelJob()
	}

	return nil
}

func (m *Manager) scheduleJob(ctx context.Context, next *rules.NextState) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// if the same state is already scheduled for an earlier time, don't schedule it again.
	if m.task != nil {
		if m.task.nextState.State == next.State &&
			m.task.firesNoLaterThan(next.Delay) {
			return
		}

		// we will replace the running job, so cancel the old one
		m.task.job.Cancel()
	}

	m.task = newTask(m.api, next)
	m.task.job = scheduler.Schedule(ctx, m.task, next.Delay)

	if next.Delay > 0 {
		m.loggers.Log(logger.Queued, next)
	}
}

func (m *Manager) cancelJob() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.task != nil {
		m.task.job.Cancel()
	}
}

func (m *Manager) processResult() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.task == nil {
		return nil
	}

	completed, err := m.task.job.Result()
	if !completed {
		return nil
	}

	if err == nil {
		m.loggers.Log(logger.Done, m.task.nextState)
	} else if errors.Is(err, scheduler.ErrCanceled) {
		m.loggers.Log(logger.Canceled, m.task.nextState)
		err = nil
	}
	// TODO: reschedule task if it failed?

	m.task = nil

	return err
}

func (m *Manager) Scheduled() (next *rules.NextState, scheduled bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.task != nil {
		next = m.task.nextState
		scheduled = true
	}
	return
}

func (m *Manager) ReportTask() (report string, scheduled bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.task != nil {
		report = m.task.Report()
		scheduled = true
	}
	return
}

type Managers []*Manager

func (m Managers) GetScheduled() (states []*rules.NextState) {
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
