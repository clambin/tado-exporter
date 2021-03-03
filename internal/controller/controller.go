package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/autoaway"
	"github.com/clambin/tado-exporter/internal/controller/overlaylimit"
	"github.com/clambin/tado-exporter/internal/controller/registry"
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// Controller object for tado-controller.
type Controller struct {
	// Configuration *configuration.ControllerConfiguration

	registry *registry.Registry
	autoAway *autoaway.AutoAway
	limiter  *overlaylimit.OverlayLimit
}

// New creates a new Controller object
func New(tadoUsername, tadoPassword, tadoClientSecret string, cfg *configuration.ControllerConfiguration) (controller *Controller, err error) {
	controller = &Controller{
		registry: &registry.Registry{
			API: &tado.APIClient{
				HTTPClient:   &http.Client{},
				Username:     tadoUsername,
				Password:     tadoPassword,
				ClientSecret: tadoClientSecret,
			},
		},
		// Configuration: cfg,
	}

	if cfg != nil && cfg.AutoAwayRules != nil {
		controller.autoAway = &autoaway.AutoAway{
			Updates:  controller.registry.Register(),
			Registry: controller.registry,
			Rules:    *cfg.AutoAwayRules,
		}
		go func() {
			controller.autoAway.Run()
		}()
	}

	if cfg != nil && cfg.OverlayLimitRules != nil {
		controller.limiter = &overlaylimit.OverlayLimit{
			Updates:  controller.registry.Register(),
			Registry: controller.registry,
			Rules:    *cfg.OverlayLimitRules,
		}
		go func() {
			controller.limiter.Run()
		}()
	}

	if cfg != nil && cfg.TadoBot.Enabled {
		callbacks := map[string]tadobot.CommandFunc{
			"rooms": controller.doRooms,
			"users": controller.doUsers,
			// "rules":        controller.doRules,
			// "autoaway":     controller.doRulesAutoAway,
			// "limitoverlay": controller.doRulesLimitOverlay,
			// "set":          controller.doSetTemperature,
		}
		if controller.registry.TadoBot, err = tadobot.Create(cfg.TadoBot.Token.Value, callbacks); err == nil {
			go func() {
				controller.registry.TadoBot.Run()
			}()
		} else {
			log.WithField("err", "failed to start TadoBot")
			controller.registry.TadoBot = nil
		}
	}

	return
}

// Run runs one update
func (controller *Controller) Run() (err error) {
	err = controller.registry.Run()

	log.WithField("err", err).Debug("Run")

	return
}

// Stop terminates all components
func (controller *Controller) Stop() (err error) {
	controller.registry.Stop()
	return
}
