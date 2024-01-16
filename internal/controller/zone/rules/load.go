package rules

import (
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
)

func LoadZoneRules(cfg configuration.ZoneConfiguration, update poller.Update, logger *slog.Logger) (rules.Rules, error) {
	id, ok := update.GetZoneID(cfg.Name)
	if !ok {
		return nil, errors.New("invalid zone name: " + cfg.Name)
	}

	if !cfg.Rules.IsActive() {
		return rules.Rules{}, nil
	}

	r := make(rules.Rules, 1, 4)
	r[0], _ = LoadHomeAwayRule(id, cfg.Name, update, logger)

	if cfg.Rules.AutoAway.IsActive() {
		rule, err := LoadAutoAwayRule(id, cfg.Name, cfg.Rules.AutoAway, update, logger)
		if err != nil {
			return nil, fmt.Errorf("invalid autoAway rule config: %w", err)
		}
		r = append(r, &rule)
	}

	if cfg.Rules.LimitOverlay.IsActive() {
		rule, _ := LoadLimitOverlay(id, cfg.Name, cfg.Rules.LimitOverlay, update, logger)
		r = append(r, &rule)
	}

	if cfg.Rules.NightTime.IsActive() {
		rule, _ := LoadNightTime(id, cfg.Name, cfg.Rules.NightTime, update, logger)
		r = append(r, &rule)
	}
	return r, nil
}
