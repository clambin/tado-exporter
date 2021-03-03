package registry

import (
	"fmt"
	"sort"
)

func (registry *Registry) GetUsers() (output []string) {
	for _, device := range registry.tadoData.MobileDevice {
		if device.Settings.GeoTrackingEnabled {
			state := "away"
			if device.Location.AtHome {
				state = "home"
			}
			if device.Location.Stale {
				state += " (stale)"
			}
			output = append(output, fmt.Sprintf("%s: %s", device.Name, state))
		}
	}
	sort.Strings(output)
	return
}

func (registry *Registry) GetRooms() (output []string) {
	for zoneID, zoneInfo := range registry.tadoData.ZoneInfo {
		mode := ""
		if zoneInfo.Overlay.Type == "MANUAL" &&
			zoneInfo.Overlay.Setting.Type == "HEATING" {
			mode = " MANUAL"
		}
		output = append(output, fmt.Sprintf("%s: %.1fºC (target: %.1fºC%s)",
			registry.tadoData.Zone[zoneID].Name,
			zoneInfo.SensorDataPoints.Temperature.Celsius,
			zoneInfo.Setting.Temperature.Celsius,
			mode,
		))
	}
	sort.Strings(output)
	return
}
