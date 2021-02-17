package controller

import "fmt"

func (controller *Controller) doUsers() (responses []string) {
	for _, device := range controller.MobileDevices {
		if device.Settings.GeoTrackingEnabled {
			state := "away"
			if device.Location.AtHome {
				state = "home"
			}
			responses = append(responses, fmt.Sprintf("%s: %s", device.Name, state))
		}
	}
	return
}

func (controller *Controller) doRooms() (responses []string) {
	return
}
