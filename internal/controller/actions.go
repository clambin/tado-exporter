package controller

import "github.com/clambin/tado-exporter/pkg/tado"

type action struct {
	Overlay           bool
	ZoneID            int
	TargetTemperature float64
}

// runAction execute an action required by autoAway/overlayLimit rules
//
// We split this out for two reasons: 1. it makes the code more readable
// and 2. makes it easier to write unit tests
//
// Currently supports creating & deleting a fixed overlay
func (controller *Controller) runAction(action action) error {
	var (
		err      error
		zoneInfo *tado.ZoneInfo
	)

	if action.Overlay == false {
		// Are we currently in overlay?
		if zoneInfo, err = controller.GetZoneInfo(action.ZoneID); err == nil {
			if zoneInfo.Overlay.Type == "MANUAL" {
				// Delete the overlay
				err = controller.DeleteZoneOverlay(action.ZoneID)
			}
		}
	} else {
		// Set the overlay
		err = controller.SetZoneOverlay(action.ZoneID, action.TargetTemperature)
	}

	return err
}
