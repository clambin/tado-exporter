package controller

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/controller/commands"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	"github.com/clambin/tado-exporter/poller"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Controller object for tado-controller
type Controller struct {
	tado.API
	zoneManagers zonemanager.Managers
	cmds         *commands.Manager
	updates      chan *poller.Update
	poller       poller.Poller
}

// New creates a new Controller object
func New(API tado.API, cfg *configuration.ControllerConfiguration, tadoBot slackbot.SlackBot, p poller.Poller) (c *Controller) {
	c = &Controller{
		API:     API,
		updates: make(chan *poller.Update, 1),
		poller:  p,
	}

	for _, zoneCfg := range cfg.ZoneConfig {
		c.zoneManagers = append(c.zoneManagers, zonemanager.New(API, p, tadoBot, zoneCfg))
	}

	if tadoBot != nil {
		c.cmds = commands.New(API, tadoBot, p, c.zoneManagers)
	}

	return c
}

// Run the controller
func (c *Controller) Run(ctx context.Context, interval time.Duration) {
	log.Info("controller started")

	wg := sync.WaitGroup{}
	wg.Add(len(c.zoneManagers))
	for _, mgr := range c.zoneManagers {
		go func(m *zonemanager.Manager) {
			m.Run(ctx, interval)
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
	log.Info("controller stopped")
}
