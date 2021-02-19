package controller

import (
	"fmt"
	"sort"
	"time"
)

func (controller *Controller) doUsers() [][]string {
	output := make([]string, 0)
	for _, device := range controller.proxy.MobileDevice {
		if device.Settings.GeoTrackingEnabled {
			state := "away"
			if device.Location.AtHome {
				state = "home"
			}
			output = append(output, fmt.Sprintf("%s: %s", device.Name, state))
		}
	}
	sort.Strings(output)
	return [][]string{output}
}

func (controller *Controller) doRooms() [][]string {
	output := make([]string, 0)
	for zoneID, zoneInfo := range controller.proxy.ZoneInfo {
		mode := ""
		if zoneInfo.Overlay.Type == "MANUAL" &&
			zoneInfo.Overlay.Setting.Type == "HEATING" {
			mode = " MANUAL"
		}
		output = append(output, fmt.Sprintf("%s: %.1fºC (target: %.1fºC%s)",
			controller.proxy.Zone[zoneID].Name,
			zoneInfo.SensorDataPoints.Temperature.Celsius,
			zoneInfo.Setting.Temperature.Celsius,
			mode,
		))
	}
	sort.Strings(output)
	return [][]string{output}
}

func (controller *Controller) doRules() (responses [][]string) {
	awayResponses := controller.doRulesAutoAway()
	limitResponses := controller.doRulesLimitOverlay()

	responses = append(responses, awayResponses[0])
	responses = append(responses, limitResponses[0])
	return
}

func (controller *Controller) doRulesAutoAway() [][]string {
	output := make([]string, 0)
	for _, entry := range controller.AutoAwayInfo {
		var response string
		switch entry.state {
		case autoAwayStateUndetermined:
			response = "undetermined"
		case autoAwayStateHome:
			response = "home"
		case autoAwayStateAway:
			response = "away. will set " + entry.Zone.Name + " to manual in " +
				entry.ActivationTime.Sub(time.Now()).Round(1*time.Minute).String()
		case autoAwayStateExpired:
			response = "away. " + entry.Zone.Name + " will be set to manual"
		case autoAwayStateReported:
			response = "away. " + entry.Zone.Name + " is set to manual"
		}
		output = append(output, entry.MobileDevice.Name+" is "+response)
	}
	sort.Strings(output)
	return [][]string{output}
}

func (controller *Controller) doRulesLimitOverlay() [][]string {
	output := make([]string, 0)
	for zoneID, entry := range controller.Overlays {
		output = append(output,
			controller.proxy.Zone[zoneID].Name+" will be reset to auto in "+entry.Sub(time.Now()).Round(1*time.Minute).String(),
		)
	}
	if len(output) > 0 {
		sort.Strings(output)
	} else {
		output = []string{"No rooms in manual control"}
	}
	return [][]string{output}
}
