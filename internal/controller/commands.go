package controller

import (
	"fmt"
	"sort"
)

func (controller *Controller) doUsers() (responses []string) {
	for _, device := range controller.proxy.MobileDevice {
		if device.Settings.GeoTrackingEnabled {
			state := "away"
			if device.Location.AtHome {
				state = "home"
			}
			responses = append(responses, fmt.Sprintf("%s: %s", device.Name, state))
		}
	}
	sort.Strings(responses)
	return
}

func (controller *Controller) doRooms() (responses []string) {
	for zoneID, zoneInfo := range controller.proxy.ZoneInfo {
		mode := ""
		if zoneInfo.Overlay.Type == "MANUAL" &&
			zoneInfo.Overlay.Setting.Type == "HEATING" {
			mode = " MANUAL"
		}
		responses = append(responses, fmt.Sprintf("%s: %.1fºC (target: %.1fºC%s)",
			controller.zoneName(zoneID),
			zoneInfo.SensorDataPoints.Temperature.Celsius,
			zoneInfo.Setting.Temperature.Celsius,
			mode,
		))
	}
	sort.Strings(responses)
	return
}
