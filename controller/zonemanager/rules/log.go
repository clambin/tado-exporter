package rules

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"golang.org/x/exp/slog"
)

func log(next TargetState, update *poller.Update) {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	zoneInfo := update.ZoneInfo[next.ZoneID]
	groups := []any{
		slogZone("zone", next),
		"next", next,
		slogZoneInfo("zoneInfo", zoneInfo),
	}
	for _, device := range update.UserInfo {
		groups = append(groups, slogDevice("device", device))
	}
	slog.Debug("next state evaluated", groups...)
}

func slogZone(name string, next TargetState) slog.Attr {
	return slog.Group(name,
		slog.Int("id", next.ZoneID),
		slog.String("name", next.ZoneName),
	)
}

func slogZoneInfo(name string, zoneInfo tado.ZoneInfo) slog.Attr {
	attribs := []slog.Attr{
		slog.String("power", zoneInfo.Setting.Power),
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

func slogDevice(name string, device tado.MobileDevice) slog.Attr {
	return slog.Group(name,
		slog.Int("id", device.ID),
		slog.String("name", device.Name),
		slog.Bool("home", device.Location.AtHome),
		slog.Bool("geotracked", device.Settings.GeoTrackingEnabled),
	)
}
