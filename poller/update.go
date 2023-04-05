package poller

import (
	"fmt"
	"github.com/clambin/tado"
	"golang.org/x/exp/slog"
)

type Update struct {
	Zones       map[int]tado.Zone
	ZoneInfo    map[int]tado.ZoneInfo
	UserInfo    MobileDevices
	WeatherInfo tado.WeatherInfo
	Home        bool
}

func (update Update) GetZoneID(name string) (int, bool) {
	for zoneID, zone := range update.Zones {
		if zone.Name == name {
			return zoneID, true
		}
	}
	return 0, false
}

func (update Update) GetUserID(name string) (int, bool) {
	for userID, user := range update.UserInfo {
		if user.Name == name {
			return userID, true
		}
	}
	return 0, false
}

type MobileDevices map[int]tado.MobileDevice

func (m MobileDevices) LogValue() slog.Value {
	var loggedDevices []slog.Attr
	for idx, device := range m {
		loggedDevices = append(loggedDevices,
			slog.Group(fmt.Sprintf("device_%d", idx),
				slog.Int("id", device.ID),
				slog.String("name", device.Name),
				slog.Bool("geotracked", device.Settings.GeoTrackingEnabled),
				slog.Group("location",
					slog.Bool("home", device.Location.AtHome),
					slog.Bool("stale", device.Location.Stale),
				),
			),
		)
	}
	return slog.GroupValue(loggedDevices...)
}
