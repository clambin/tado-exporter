package rules

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"strings"
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

	var home []string
	var away []string
	for _, id := range a.mobileDeviceID {
		if entry, exists := update.UserInfo[id]; exists {
			if entry.IsHome() == tado.DeviceAway {
				away = append(away, entry.Name)
			} else {
				home = append(home, entry.Name)
			}
		}
	}

	state := update.ZoneInfo[a.zoneID].GetState()
	if state == tado.ZoneStateOff {
		if len(home) != 0 {
			next = &NextState{
				ZoneID:       a.zoneID,
				ZoneName:     a.zoneName,
				State:        tado.ZoneStateAuto,
				Delay:        0,
				ActionReason: makeReason(home, "home"),
				CancelReason: makeReason(home, "away"),
			}
		}
	} else {
		if len(home) == 0 {
			next = &NextState{
				ZoneID:       a.zoneID,
				ZoneName:     a.zoneName,
				State:        tado.ZoneStateOff,
				Delay:        a.config.Delay,
				ActionReason: makeReason(away, "away"),
				CancelReason: makeReason(away, "home"),
			}
		}
	}
	return
}

func makeReason(users []string, state string) string {
	var verb string
	if len(users) == 1 {
		verb = "is"
	} else {
		verb = "are"
	}
	return fmt.Sprintf("%s %s %s", strings.Join(users, ", "), verb, state)
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
