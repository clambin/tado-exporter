package zonemanager

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager/logger"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/pkg/scheduler"
	"github.com/clambin/tado-exporter/poller"
	"golang.org/x/exp/slog"
	"sync"
)

type Manager struct {
	evaluator rules.Evaluator
	task      *Task
	api       TadoSetter
	loggers   logger.Loggers
	poller    poller.Poller
	notifier  chan struct{}
	lock      sync.RWMutex
}

func New(api TadoSetter, p poller.Poller, bot slackbot.SlackBot, cfg rules.ZoneConfig) *Manager {
	loggers := logger.Loggers{&logger.StdOutLogger{}}
	if bot != nil {
		loggers = append(loggers, &logger.SlackLogger{Bot: bot})
	}

	return &Manager{
		evaluator: rules.Evaluator{Config: &cfg},
		api:       api,
		loggers:   loggers,
		poller:    p,
		notifier:  make(chan struct{}, 1),
	}
}

func (m *Manager) Run(ctx context.Context) {
	ch := m.poller.Register()
	defer m.poller.Unregister(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case update := <-ch:
			if err := m.processUpdate(ctx, update); err != nil {
				slog.Error("failed to process tado update", "err", err, "zone", m.evaluator.Config.Zone)
			}
		case <-m.notifier:
			if err := m.processResult(); err != nil {
				slog.Error("failed to set next state", "err", err, "zone", m.evaluator.Config.Zone)
			}
		}
	}
}

func (m *Manager) processUpdate(ctx context.Context, update *poller.Update) error {
	next, err := m.evaluator.Evaluate(update)
	if err != nil {
		return fmt.Errorf("failed to evaluate rules: %w", err)
	}

	if next.Action {
		slogJob(next, update)
		m.scheduleJob(ctx, next)
	} else {
		m.cancelJob(next)
	}

	return nil
}

func slogJob(next rules.TargetState, update *poller.Update) {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	zoneGroup := []slog.Attr{slog.Group("settings",
		slog.String("power", update.ZoneInfo[next.ZoneID].Setting.Power),
		slog.Float64("temperature", update.ZoneInfo[next.ZoneID].Setting.Temperature.Celsius),
	)}
	if update.ZoneInfo[next.ZoneID].Overlay.GetMode() != tado.NoOverlay {
		zoneGroup = append(zoneGroup, slog.Group("overlay",
			slog.String("power", update.ZoneInfo[next.ZoneID].Overlay.Setting.Power),
			slog.Float64("temperature", update.ZoneInfo[next.ZoneID].Overlay.Setting.Temperature.Celsius),
		))
	}
	slog.Debug("scheduling job", "next", next, slog.Group("zoneState", zoneGroup...))
}

func (m *Manager) scheduleJob(ctx context.Context, next rules.TargetState) {
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
	m.task.job = scheduler.ScheduleWithNotification(ctx, m.task, next.Delay, m.notifier)
	if next.Delay > 0 {
		m.loggers.Log(logger.Queued, next)
	}
}

func (m *Manager) cancelJob(next rules.TargetState) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.task != nil {
		nextState := m.task.nextState
		nextState.Reason = next.Reason
		m.task.job.Cancel()
		m.loggers.Log(logger.Canceled, nextState)
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
		err = nil
	}
	// TODO: reschedule task if it failed?

	m.task = nil
	return err
}

func (m *Manager) GetScheduled() (rules.TargetState, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.task != nil {
		return m.task.nextState, true
	}
	return rules.TargetState{}, false
}

func (m *Manager) ReportTask() (string, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.task != nil {
		return m.task.Report(), true
	}
	return "", false
}

type Managers []*Manager

func (m Managers) GetScheduled() []rules.TargetState {
	var states []rules.TargetState
	for _, mgr := range m {
		if state, scheduled := mgr.GetScheduled(); scheduled {
			states = append(states, state)
		}
	}
	return states
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
