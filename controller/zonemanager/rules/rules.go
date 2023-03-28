package rules

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"golang.org/x/exp/slog"
)

type Evaluator struct {
	Config *ZoneConfig
	rules  []Rule
}

var _ Rule = &Evaluator{}

type Rule interface {
	Evaluate(*poller.Update) (TargetState, error)
}

func (e *Evaluator) Evaluate(update *poller.Update) (TargetState, error) {
	var targetState TargetState
	if err := e.load(update); err != nil {
		return targetState, err
	}

	var targetStates TargetStates
	for _, rule := range e.rules {
		next, err := rule.Evaluate(update)
		if err != nil {
			return targetState, err
		}
		targetStates = append(targetStates, next)
	}

	next := targetStates.GetNextState()
	e.slog(next, update)

	return next, nil
}

func (e *Evaluator) load(update *poller.Update) error {
	if len(e.rules) > 0 {
		return nil
	}

	zoneID, ok := update.GetZoneID(e.Config.Zone)
	if !ok {
		return fmt.Errorf("invalid zone found in config file: %s", e.Config.Zone)
	}

	for _, rawRule := range e.Config.Rules {
		switch rawRule.Kind {
		case AutoAway:
			e.rules = append(e.rules, &AutoAwayRule{
				ZoneID:   zoneID,
				ZoneName: e.Config.Zone,
				Delay:    rawRule.Delay,
				Users:    rawRule.Users,
			})
		case LimitOverlay:
			e.rules = append(e.rules, &LimitOverlayRule{
				zoneID:   zoneID,
				zoneName: e.Config.Zone,
				delay:    rawRule.Delay,
			})
		case NightTime:
			e.rules = append(e.rules, &NightTimeRule{
				zoneID:    zoneID,
				zoneName:  e.Config.Zone,
				timestamp: rawRule.Timestamp,
			})
		}
	}

	return nil
}

func (e *Evaluator) slog(next TargetState, update *poller.Update) {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	zoneInfo := update.ZoneInfo[next.ZoneID]
	groups := []any{
		slog.Group("zone",
			slog.Int("id", next.ZoneID),
			slog.String("name", next.ZoneName),
		),
		"next", next,
		slogZoneInfo("zoneInfo", zoneInfo),
	}
	for id, device := range update.UserInfo {
		groups = append(groups,
			slog.Group("device",
				slog.Int("id", id),
				slog.String("name", device.Name),
				slog.Bool("home", device.Location.AtHome),
				slog.Bool("geotracked", device.Settings.GeoTrackingEnabled),
			),
		)
	}
	slog.Debug("next state evaluated", groups...)
}

func slogZoneInfo(name string, zoneInfo tado.ZoneInfo) slog.Attr {
	attribs := []slog.Attr{
		slog.String("power", zoneInfo.Setting.Power),
	}
	if zoneInfo.Overlay.Type != "" {
		attribs = append(attribs, slog.Group("overlay",
			slog.String("type", zoneInfo.Overlay.Type),
			slog.Group("setting",
				slog.String("type", zoneInfo.Overlay.Termination.Type),
				slog.String("subtype", zoneInfo.Overlay.Termination.TypeSkillBasedApp),
			),
			slog.Group("termination",
				slog.String("type", zoneInfo.Overlay.Setting.Type),
				slog.String("power", zoneInfo.Overlay.Setting.Power),
			),
		))
	}
	return slog.Group(name, attribs...)
}
