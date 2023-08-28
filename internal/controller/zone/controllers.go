package zone

import (
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
)

type Controllers []*Controller

func (c Controllers) GetScheduled() []rules.Action {
	var states []rules.Action
	for _, controller := range c {
		if state, scheduled := controller.GetScheduled(); scheduled {
			states = append(states, state)
		}
	}
	return states
}

func (c Controllers) ReportTasks() ([]string, bool) {
	var tasks []string
	for _, controller := range c {
		if task, ok := controller.ReportTask(); ok {
			tasks = append(tasks, task)
		}
	}
	return tasks, len(tasks) > 0
}
