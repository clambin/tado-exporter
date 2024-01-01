package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/rules/home"
	"github.com/clambin/tado-exporter/internal/poller"
)

func LoadHomeRules(cfg configuration.HomeConfiguration, update poller.Update) (Rules, error) {
	var r Rules

	if cfg.AutoAway.IsActive() {
		rule, err := home.LoadAutoAwayRule(cfg.AutoAway, update)
		if err != nil {
			return nil, fmt.Errorf("invalid autoAway rule config: %w", err)
		}
		r = append(r, &rule)
	}

	return r, nil
}
