package rules

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"strings"
	"time"
)

type AutoAwayRule struct {
	zoneID          int
	zoneName        string
	delay           time.Duration
	users           []string
	mobileDeviceIDs []int
}

var _ Rule = &AutoAwayRule{}

func (a *AutoAwayRule) Evaluate(update *poller.Update) (NextState, error) {
	var next NextState
	if err := a.load(update); err != nil {
		return next, err
	}

	var home []string
	var away []string
	for _, id := range a.mobileDeviceIDs {
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
			next = NextState{
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
			next = NextState{
				ZoneID:       a.zoneID,
				ZoneName:     a.zoneName,
				State:        tado.ZoneStateOff,
				Delay:        a.delay,
				ActionReason: makeReason(away, "away"),
				CancelReason: makeReason(away, "home"),
			}
		}
	}
	return next, nil
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
	if len(a.mobileDeviceIDs) > 0 {
		return nil
	}

	for _, user := range a.users {
		if userID, ok := update.GetUserID(user); ok {
			a.mobileDeviceIDs = append(a.mobileDeviceIDs, userID)
		} else {
			return fmt.Errorf("invalid user: %s", user)
		}
	}

	return nil
}
