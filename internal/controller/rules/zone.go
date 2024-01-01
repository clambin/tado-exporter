package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/rules/zone"
	"github.com/clambin/tado-exporter/internal/poller"
)

func LoadZoneRules(cfg configuration.ZoneConfiguration, update poller.Update) (Rules, error) {
	var r Rules

	id, ok := update.GetZoneID(cfg.Name)
	if !ok {
		return nil, fmt.Errorf("invalid zone name: %s", cfg.Name)
	}

	if cfg.Rules.AutoAway.IsActive() {
		rule, err := zone.LoadAutoAwayRule(id, cfg.Name, cfg.Rules.AutoAway, update)
		if err != nil {
			return nil, fmt.Errorf("invalid autoAway rule config: %w", err)
		}
		r = append(r, &rule)
	}

	if cfg.Rules.LimitOverlay.IsActive() {
		rule, err := zone.LoadLimitOverlay(id, cfg.Name, cfg.Rules.LimitOverlay, update)
		if err != nil {
			return nil, fmt.Errorf("invalid limitOverlay config: %w", err)
		}
		r = append(r, &rule)
	}

	if cfg.Rules.NightTime.IsActive() {
		rule, err := zone.LoadNightTime(id, cfg.Name, cfg.Rules.NightTime, update)
		if err != nil {
			return nil, fmt.Errorf("invalid nightTime rule configuration: %w", err)
		}
		r = append(r, &rule)
	}
	return r, nil
}
