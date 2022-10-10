package rules

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
)

type AutoAwayRule struct {
	zoneID         int
	zoneName       string
	config         *configuration.ZoneAutoAway
	mobileDeviceID []int
}

var _ Rule = &AutoAwayRule{}

func (a *AutoAwayRule) Evaluate(update *poller.Update) (next *NextState, err error) {
	if err = a.load(update); err != nil {
		return nil, err
	}
	var home bool
	for _, id := range a.mobileDeviceID {
		if entry, exists := update.UserInfo[id]; exists {
			if entry.IsHome() != tado.DeviceAway {
				home = true
				break
			}
		}
	}
	state := update.ZoneInfo[a.zoneID].GetState()

	if !home && state != tado.ZoneStateOff {
		next = &NextState{
			ZoneID:       a.zoneID,
			ZoneName:     a.zoneName,
			State:        tado.ZoneStateOff,
			Delay:        a.config.Delay,
			ActionReason: "user(s) is/are away",
			CancelReason: "user(s) is/are home",
		}
	} else if home && state == tado.ZoneStateOff {
		next = &NextState{
			ZoneID:       a.zoneID,
			ZoneName:     a.zoneName,
			State:        tado.ZoneStateAuto,
			Delay:        0,
			ActionReason: "user(s) is/are home",
			CancelReason: "user(s) is/are away",
		}
	}

	return
}

func (a *AutoAwayRule) load(update *poller.Update) error {
	if len(a.mobileDeviceID) > 0 {
		return nil
	}

	var userIDs []int
	for _, user := range a.config.Users {
		if userID, _, found := update.LookupUser(user.MobileDeviceID, user.MobileDeviceName); found {
			userIDs = append(userIDs, userID)
		} else {
			return fmt.Errorf("invalid user found in config file: zoneID: %d, zoneName: %s", user.MobileDeviceID, user.MobileDeviceName)
		}

	}
	a.mobileDeviceID = userIDs

	return nil
}
