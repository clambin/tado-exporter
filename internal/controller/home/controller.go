package home

import (
	homeRules "github.com/clambin/tado-exporter/internal/controller/home/rules"
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/processor"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

type Controller struct {
	*processor.Processor
}

func New(tadoClient action.TadoSetter, p poller.Poller, bot notifier.SlackSender, configuration configuration.HomeConfiguration, logger *slog.Logger) *Controller {
	loader := func(update poller.Update) (rules.Evaluator, error) {
		return homeRules.LoadHomeRules(configuration, update, logger)
	}

	return &Controller{
		Processor: processor.New(tadoClient, p, bot, loader, logger),
	}
}
