package zone

import (
	"github.com/clambin/tado"
	"log/slog"
)

var _ slog.LogValuer = zoneLogger{}

type zoneLogger tado.ZoneInfo

func (z zoneLogger) LogValue() slog.Value {
	zoneGroup := make([]slog.Attr, 1, 2)

	attribs := make([]any, 1, 2)
	attribs[0] = slog.String("power", z.Setting.Power)
	if z.Setting.Power == "ON" {
		attribs = append(attribs, slog.Float64("temperature", z.Setting.Temperature.Celsius))
	}
	zoneGroup[0] = slog.Group("settings", attribs...)

	if z.Overlay.GetMode() != tado.NoOverlay {
		zoneGroup = append(zoneGroup,
			slog.Group("overlay",
				slog.Group("termination",
					slog.String("type", z.Overlay.Termination.Type),
					slog.String("subtype", z.Overlay.Termination.TypeSkillBasedApp),
				),
			))
	}
	return slog.GroupValue(zoneGroup...)
}
