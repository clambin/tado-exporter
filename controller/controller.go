package controller

import (
	"context"
	"github.com/clambin/tado-exporter/controller/commands"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/poller"
	tadoAPI "github.com/clambin/tado-exporter/tado"
	"golang.org/x/exp/slog"
	"sync"
)

// Controller object for tado-controller
type Controller struct {
	zoneManagers zonemanager.Managers
	cmds         *commands.Manager
}

// New creates a new Controller object
func New(api tadoAPI.API, cfg []rules.ZoneConfig, tadoBot slackbot.SlackBot, p poller.Poller) *Controller {
	var c Controller

	for _, zoneCfg := range cfg {
		c.zoneManagers = append(c.zoneManagers, zonemanager.New(api, p, tadoBot, zoneCfg))
	}

	if tadoBot != nil {
		c.cmds = commands.New(api, tadoBot, p, c.zoneManagers)
	}

	return &c
}

// Run the controller
func (c *Controller) Run(ctx context.Context) {
	slog.Info("controller started")

	wg := sync.WaitGroup{}
	wg.Add(len(c.zoneManagers))
	for _, mgr := range c.zoneManagers {
		go func(m *zonemanager.Manager) {
			m.Run(ctx)
			wg.Done()
		}(mgr)
	}

	if c.cmds != nil {
		wg.Add(1)
		go func() {
			c.cmds.Run(ctx)
			wg.Done()
		}()
	}

	wg.Wait()
	slog.Info("controller stopped")
}
