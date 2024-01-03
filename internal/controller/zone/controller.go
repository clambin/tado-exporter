package zone

import (
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/processor"
	rules2 "github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

type Controller struct {
	configuration configuration.ZoneConfiguration
	*processor.Processor
}

func New(tadoClient action.TadoSetter, p poller.Poller, bot notifier.SlackSender, configuration configuration.ZoneConfiguration, logger *slog.Logger) *Controller {
	loader := func(update poller.Update) (rules2.Evaluator, error) {
		return rules.LoadZoneRules(configuration, update)
	}

	return &Controller{
		configuration: configuration,
		Processor:     processor.New(tadoClient, p, bot, loader, logger),
	}
}
