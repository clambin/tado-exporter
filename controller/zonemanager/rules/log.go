package rules

import (
	"github.com/clambin/tado"
	"golang.org/x/exp/slog"
)

func slogZoneInfo(name string, zoneInfo tado.ZoneInfo) slog.Attr {
	attribs := make([]slog.Attr, 1, 3)
	attribs[0] = slog.String("power", zoneInfo.Setting.Power)
	if zoneInfo.Setting.Power == "ON" {
		attribs = append(attribs, slog.Float64("temperature", zoneInfo.Setting.Temperature.Celsius))
	}
	if zoneInfo.Overlay.Type != "" {
		attribs = append(attribs, slog.Group("overlay",
			slog.String("type", zoneInfo.Overlay.Type),
			slog.Group("setting",
				slog.String("type", zoneInfo.Overlay.Setting.Type),
				slog.String("power", zoneInfo.Overlay.Setting.Power),
			),
			slog.Group("termination",
				slog.String("type", zoneInfo.Overlay.Termination.Type),
				slog.String("subtype", zoneInfo.Overlay.Termination.TypeSkillBasedApp),
			),
		))
	}
	return slog.Group(name, attribs...)
}
