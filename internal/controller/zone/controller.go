package zone

import (
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/processor"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	zoneRules "github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

type Controller struct {
	*processor.Processor
}

func New(tadoClient action.TadoClient, p poller.Poller, bot notifier.SlackSender, configuration configuration.ZoneConfiguration, logger *slog.Logger) *Controller {
	loader := func(update poller.Update) (rules.Evaluator, error) {
		return zoneRules.LoadZoneRules(configuration, update, logger)
	}

	return &Controller{
		Processor: processor.New(tadoClient, p, bot, loader, logger),
	}
}
