package overlaylimit

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/actions"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"time"
)

type OverlayLimit struct {
	actions.Actions

	Updates     chan *scheduler.TadoData
	Slack       tadobot.PostChannel
	Rules       []*configuration.OverlayLimitRule
	zoneDetails map[int]zoneDetails
}

type zoneDetails struct {
	zone        tado.Zone
	rule        configuration.OverlayLimitRule
	isOverlay   bool
	expiryTimer time.Time
}

// Run waits for updates data from the scheduler and evaluates configured overlayLimit rules
func (overlayLimit *OverlayLimit) Run() {
	for tadoData := range overlayLimit.Updates {
		if tadoData == nil {
			break
		}
		if err := overlayLimit.process(tadoData); err != nil {
			log.WithField("err", err).Warning("failed to process autoAway rules")
		}
	}
}

// process sets the state of each zone, checks if have expired and sets them back to auto mode
func (overlayLimit *OverlayLimit) process(tadoData *scheduler.TadoData) (err error) {
	var actionList []actions.Action

	_ = overlayLimit.updateInfo(tadoData)
	if actionList, err = overlayLimit.getActions(); err == nil {
		if err = overlayLimit.RunActions(actionList); err != nil {
			log.WithField("err", err).Warning("failed to set action")
		}
	}

	return
}

// updateInfo updates the details on any monitored zone
func (overlayLimit *OverlayLimit) updateInfo(tadoData *scheduler.TadoData) (err error) {
	if overlayLimit.zoneDetails == nil {
		overlayLimit.initZoneDetails(tadoData)
	}

	for id, details := range overlayLimit.zoneDetails {
		if zoneInfo, ok := tadoData.ZoneInfo[id]; ok {
			if zoneInfo.Overlay.Type == "MANUAL" &&
				zoneInfo.Overlay.Setting.Type == "HEATING" &&
				zoneInfo.Overlay.Termination.Type == "MANUAL" {
				if details.isOverlay == false {

					// zone in overlay. record when we need to disable the overlay

					log.WithField("overlay", zoneInfo.Overlay.String()).Debug("overlay found")

					details.isOverlay = true
					details.expiryTimer = time.Now().Add(details.rule.MaxTime)
					overlayLimit.zoneDetails[id] = details

					log.WithFields(log.Fields{
						"zoneID":   details.zone.ID,
						"zoneName": details.zone.Name,
						"expiry":   details.expiryTimer,
					}).Info("new zone in overlay")
					// notify via slack if needed
					if overlayLimit.Slack != nil {
						overlayLimit.Slack <- []slack.Attachment{
							{
								Color: "good",
								Title: "Manual temperature setting detected in zone " + details.zone.Name,
							},
						}
					}
				}
			} else if details.isOverlay == true {
				// Zone is not in overlay

				details.isOverlay = false
				overlayLimit.zoneDetails[id] = details

				log.WithFields(log.Fields{
					"zoneID":   details.zone.ID,
					"zoneName": details.zone.Name,
				}).Info("zone no longer in overlay")
			}
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
			zone: *zone,
			rule: *rule,
		}
	}
}

// getActions deletes any overlays that have expired
func (overlayLimit *OverlayLimit) getActions() (actionList []actions.Action, err error) {
	for id, details := range overlayLimit.zoneDetails {
		if details.isOverlay && time.Now().After(details.expiryTimer) {
			actionList = append(actionList, actions.Action{
				Overlay: false,
				ZoneID:  id,
			})
			log.WithField("zoneID", id).Info("expiring overlay in zone")
			// Technically not needed (next run will do this automatically, but facilitates unit testing
			details.isOverlay = false
			overlayLimit.zoneDetails[id] = details
			// notify via slack if needed
			if overlayLimit.Slack != nil {
				overlayLimit.Slack <- []slack.Attachment{
					{
						Color: "good",
						Title: "Disabling manual temperature setting in zone " + details.zone.Name,
					},
				}
			}
		}
	}
	return
}
