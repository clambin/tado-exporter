package controller

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
)

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
		ok       bool
		zoneInfo *tado.ZoneInfo
	)

	if action.Overlay == false {
		// Are we currently in overlay?
		if zoneInfo, ok = controller.proxy.ZoneInfo[action.ZoneID]; ok {
			if zoneInfo.Overlay.Type == "MANUAL" {
				// Delete the overlay
				err = controller.proxy.DeleteZoneOverlay(action.ZoneID)
			} else {
				// TODO: does this ever happen?
				log.WithField("type", zoneInfo.Overlay.Type).Info("not a manual overlay type. not deleting")
			}
		}
	} else {
		// Set the overlay
		err = controller.proxy.SetZoneOverlay(action.ZoneID, action.TargetTemperature)
	}

	return err
}
