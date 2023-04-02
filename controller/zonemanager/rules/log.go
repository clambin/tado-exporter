package rules

import (
	"github.com/clambin/tado"
	"golang.org/x/exp/slog"
)

var _ slog.LogValuer = zoneInfo{}

type zoneInfo tado.ZoneInfo

func (z zoneInfo) LogValue() slog.Value {
	attribs := make([]slog.Attr, 1, 2)

	settings := make([]slog.Attr, 1, 2)
	settings[0] = slog.String("power", z.Setting.Power)
	if z.Setting.Power == "ON" {
		settings = append(settings, slog.Float64("temperature", z.Setting.Temperature.Celsius))
	}
	attribs[0] = slog.Group("settings", settings...)

	if z.Overlay.Type != "" {
		attribs = append(attribs, slog.Group("overlay",
			slog.String("type", z.Overlay.Type),
			slog.Group("termination",
				slog.String("type", z.Overlay.Termination.Type),
				slog.String("subtype", z.Overlay.Termination.TypeSkillBasedApp),
			),
		))
	}
	return slog.GroupValue(attribs...)
}
