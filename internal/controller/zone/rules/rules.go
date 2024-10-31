package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

func LoadZoneRules(cfg configuration.ZoneConfiguration, update poller.Update, logger *slog.Logger) (rules.Rules, error) {
	zone, err := update.GetZone(cfg.Name)
	if err != nil {
		return nil, err
	}

	if !cfg.Rules.IsActive() {
		return rules.Rules{}, nil
	}

	r := make(rules.Rules, 1, 4)
	r[0], _ = LoadHomeAwayRule(*zone.Zone.Id, cfg.Name, update, logger)

	if cfg.Rules.AutoAway.IsActive() {
		rule, err := LoadAutoAwayRule(*zone.Id, cfg.Name, cfg.Rules.AutoAway, update, logger)
		if err != nil {
			return nil, fmt.Errorf("invalid autoAway rule config: %w", err)
		}
		r = append(r, &rule)
	}

	if cfg.Rules.LimitOverlay.IsActive() {
		rule, _ := LoadLimitOverlay(*zone.Id, cfg.Name, cfg.Rules.LimitOverlay, update, logger)
		r = append(r, &rule)
	}

	if cfg.Rules.NightTime.IsActive() {
		rule, _ := LoadNightTime(*zone.Id, cfg.Name, cfg.Rules.NightTime, update, logger)
		r = append(r, &rule)
	}
	return r, nil
}
