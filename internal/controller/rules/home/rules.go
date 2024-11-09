package home

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

func LoadHomeRules(cfg configuration.HomeConfiguration, update poller.Update, logger *slog.Logger) (rules.Rules, error) {
	var r rules.Rules

	if cfg.AutoAway.IsActive() {
		rule, err := LoadAutoAwayRule(cfg.AutoAway, update, logger)
		if err != nil {
			return nil, fmt.Errorf("invalid autoAway rule config: %w", err)
		}
		r = append(r, &rule)
	}

	return r, nil
}
