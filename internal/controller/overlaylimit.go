package controller

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"time"
)

func (controller *Controller) OverlayLimitRun() error {
	if controller.Rules.OverlayLimit == nil {
		return nil
	}

	if controller.Overlays == nil {
		controller.Overlays = make(map[int]time.Time)
	}

	var err error

	if err = controller.OverlayLimitUpdateInfo(); err == nil {
		err = controller.ExpireOverlays()
	}

	log.WithFields(log.Fields{
		"err":      err,
		"overlays": len(controller.Overlays),
	}).Debug("OverlayLimitRun")

	return err
}

func (controller *Controller) OverlayLimitUpdateInfo() error {
	var (
		err   error
		zones []tado.Zone
	)

	if zones, err = controller.GetZones(); err == nil {
		for _, overlayLimit := range *controller.Rules.OverlayLimit {
			var (
				zone     *tado.Zone
				zoneInfo *tado.ZoneInfo
			)
			if zone = getZone(zones, overlayLimit.ZoneID, overlayLimit.ZoneName); zone == nil {
				log.WithFields(log.Fields{
					"ZoneID":   overlayLimit.ZoneID,
					"ZoneName": overlayLimit.ZoneName,
				}).Warning("skipping unknown zone in OverlayLimit rule")
				continue
			}

			if zoneInfo, err = controller.GetZoneInfo(zone.ID); err == nil {
				if zoneInfo.Overlay.Type == "MANUAL" && zoneInfo.Overlay.Setting.Type == "HEATING" {
					// Zone in overlay. If we're not already tracking it, add it now
					if _, ok := controller.Overlays[zone.ID]; ok == false {
						expiry := time.Now().Add(overlayLimit.MaxTime)
						controller.Overlays[zone.ID] = expiry
						log.WithFields(log.Fields{
							"zoneID": zone.ID,
							"expiry": expiry,
						}).Info("new zone in overlay")
					}
				} else {
					// Zone is not in overlay. Remove it from the tracking map
					if _, ok := controller.Overlays[zone.ID]; ok == true {
						delete(controller.Overlays, zone.ID)
						log.WithField("zoneID", zone.ID).Info("zone no longer in overlay")
					}
				}
			}

		}
	}

	return err
}

func (controller *Controller) ExpireOverlays() error {
	var err error
	for zoneID, expiryTimer := range controller.Overlays {
		if time.Now().After(expiryTimer) {
			err = controller.DeleteZoneManualTemperature(zoneID)
			log.WithField("zoneID", zoneID).Info("expiring overlay in zone")
			// Technically not needed (next run will do this automatically, but facilitates unit testing
			delete(controller.Overlays, zoneID)
		}
	}
	return err
}
