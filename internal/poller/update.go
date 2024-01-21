package poller

import (
	"github.com/clambin/tado"
	"log/slog"
	"slices"
	"strconv"
)

type Update struct {
	WeatherInfo tado.WeatherInfo
	Zones       map[int]tado.Zone
	ZoneInfo    map[int]tado.ZoneInfo
	UserInfo    MobileDevices
	Home        IsHome
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

func (update Update) GetDeviceStatus(ids ...int) ([]string, []string) {
	if len(ids) == 0 {
		for id := range update.UserInfo {
			ids = append(ids, id)
		}
	}

	var home, away []string
	for _, id := range ids {
		if entry, exists := update.UserInfo[id]; exists {
			switch entry.IsHome() {
			case tado.DeviceHome:
				home = append(home, entry.Name)
			case tado.DeviceAway, tado.DeviceUnknown:
				away = append(away, entry.Name)
			}
		}
	}
	return home, away
}

type IsHome bool

func (i IsHome) String() string {
	if i {
		return "HOME"
	} else {
		return "AWAY"
	}
}

type MobileDevices map[int]tado.MobileDevice

func (m MobileDevices) LogValue() slog.Value {
	var loggedDevices []slog.Attr
	for _, deviceID := range m.sortedDeviceIDs() {
		device := m[deviceID]
		attribs := logDevice(device)
		loggedDevices = append(loggedDevices, slog.Group("device_"+strconv.Itoa(deviceID), attribs...))
	}
	return slog.GroupValue(loggedDevices...)
}

func (m MobileDevices) sortedDeviceIDs() []int {
	deviceIDs := make([]int, 0, len(m))
	for deviceID := range m {
		deviceIDs = append(deviceIDs, deviceID)
	}
	slices.Sort(deviceIDs)
	return deviceIDs
}

func logDevice(device tado.MobileDevice) []any {
	attribs := make([]any, 3, 4)
	attribs[0] = slog.Int("id", device.ID)
	attribs[1] = slog.String("name", device.Name)
	attribs[2] = slog.Bool("geotracked", device.Settings.GeoTrackingEnabled)
	if device.Settings.GeoTrackingEnabled {
		attribs = append(attribs,
			slog.Group("location",
				slog.Bool("home", device.Location.AtHome),
				slog.Bool("stale", device.Location.Stale),
			),
		)
	}
	return attribs
}
