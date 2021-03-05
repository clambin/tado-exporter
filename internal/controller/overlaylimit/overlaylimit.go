package overlaylimit

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"time"
)

type OverlayLimit struct {
	Updates     scheduler.UpdateChannel
	RoomSetter  chan tadosetter.RoomCommand
	Slack       tadobot.PostChannel
	Rules       []*configuration.OverlayLimitRule
	zoneDetails map[int]zoneDetails
}

// Run waits for updates data from the scheduler and evaluates configured overlayLimit rules
func (overlayLimit *OverlayLimit) Run() {
	for tadoData := range overlayLimit.Updates {
		if tadoData == nil {
			break
		}
		log.WithField("object", *tadoData).Debug("got a message")
		overlayLimit.updateInfo(tadoData)
		overlayLimit.process(tadoData)
	}
}

// updateInfo updates the details on any monitored zone
func (overlayLimit *OverlayLimit) updateInfo(tadoData *scheduler.TadoData) {
	if overlayLimit.zoneDetails == nil {
		overlayLimit.initZoneDetails(tadoData)
	}

	var (
		zoneInfo tado.ZoneInfo
		ok       bool
	)
	for id, details := range overlayLimit.zoneDetails {
		if zoneInfo, ok = tadoData.ZoneInfo[id]; !ok {
			continue
		}
		if zoneInfo.Overlay.Type == "MANUAL" &&
			zoneInfo.Overlay.Setting.Type == "HEATING" &&
			zoneInfo.Overlay.Termination.Type == "MANUAL" {
			if details.state <= zoneStateAuto {
				// zone in overlay. record when we need to disable the overlay
				log.WithField("overlay", zoneInfo.Overlay.String()).Debug("overlay found")
				details.state = zoneStateManual
				details.expiryTimer = time.Now().Add(details.rule.MaxTime)
				overlayLimit.zoneDetails[id] = details
			}
		} else if details.state > zoneStateAuto {
			// Zone is no longer in overlay
			details.state = zoneStateAuto
			overlayLimit.zoneDetails[id] = details
		}
	}
	return
}

func (overlayLimit *OverlayLimit) initZoneDetails(tadoData *scheduler.TadoData) {
	overlayLimit.zoneDetails = make(map[int]zoneDetails)

	for _, rule := range overlayLimit.Rules {
		var zone *tado.Zone

		// Rules file can contain either zone ID or Name. Retrieve the ID for each of these
		// and discard any that aren't valid.

		if zone = scheduler.LookupZone(tadoData, rule.ZoneID, rule.ZoneName); zone == nil {
			log.WithFields(log.Fields{
				"zoneID":   rule.ZoneID,
				"zoneName": rule.ZoneName,
			}).Warning("skipping unknown zone in OverlayLimit rule")
			continue
		}

		overlayLimit.zoneDetails[zone.ID] = zoneDetails{
			zone:  *zone,
			rule:  *rule,
			state: zoneStateUndetermined,
		}
	}
}

// getActions deletes any overlays that have expired
func (overlayLimit *OverlayLimit) process(_ *scheduler.TadoData) {
	for id, details := range overlayLimit.zoneDetails {
		switch details.state {
		case zoneStateManual:
			// Zone is now in overlay. Report to slack
			if overlayLimit.Slack != nil {
				overlayLimit.Slack <- []slack.Attachment{{
					Color: "good",
					Title: "new zone in overlay",
					Text:  "Manual temperature setting detected in zone " + details.zone.Name,
				}}
			}
			details.expiryTimer = time.Now().Add(details.rule.MaxTime)
			details.state = zoneStateReported
		case zoneStateReported:
			// Zone is in overlay. Do we need to reset it?
			if time.Now().After(details.expiryTimer) {
				log.WithField("id", id).Debug("expired overlay found. deleting")
				if overlayLimit.Slack != nil {
					overlayLimit.Slack <- []slack.Attachment{{
						Color: "good",
						Title: "overlay expired",
						Text:  "Disabling manual temperature setting in zone " + details.zone.Name,
					}}
				}
				overlayLimit.RoomSetter <- tadosetter.RoomCommand{
					ZoneID: id,
					Auto:   true,
				}
				details.state = zoneStateExpired
			}
		}
		overlayLimit.zoneDetails[id] = details
	}
}
