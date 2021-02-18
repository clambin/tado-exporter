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

func (controller *Controller) doRules() (responses []string) {
	awayResponses := controller.doRulesAutoAway()
	limitResponses := controller.doRulesLimitOverlay()

	responses = append(responses, awayResponses...)
	if len(awayResponses) > 0 && len(limitResponses) > 0 {
		responses = append(responses, "")
	}
	responses = append(responses, limitResponses...)
	return
}

func (controller *Controller) doRulesAutoAway() (responses []string) {
	for _, entry := range controller.AutoAwayInfo {
		var response string
		switch entry.state {
		case autoAwayStateUndetermined:
			response = "undetermined"
		case autoAwayStateHome:
			response = "home"
		case autoAwayStateAway:
			response = fmt.Sprintf("away. will set room %s to manual at %s",
				controller.zoneName(entry.ZoneID),
				entry.ActivationTime.Format("2006-01-02 15:04:05"),
			)
		case autoAwayStateExpired:
			response = "away but not yet reported"
		case autoAwayStateReported:
			response = "away & reported"
		}
		responses = append(responses,
			entry.MobileDevice.Name+": "+response,
		)
	}
	sort.Strings(responses)
	return
}

func (controller *Controller) doRulesLimitOverlay() (responses []string) {
	for zoneID, entry := range controller.Overlays {
		responses = append(responses, fmt.Sprintf("room %s will be set back to auto on %s",
			controller.zoneName(zoneID),
			entry.Format("2006-01-02 15:04:05"),
		))
	}
	if len(responses) > 0 {
		sort.Strings(responses)
	} else {
		responses = append(responses, "No rooms in manual control")
	}
	return
}
