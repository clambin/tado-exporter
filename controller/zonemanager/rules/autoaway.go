package rules

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"strings"
	"time"
)

type AutoAwayRule struct {
	ZoneID          int
	ZoneName        string
	Delay           time.Duration
	Users           []string
	MobileDeviceIDs []int
}

var _ Rule = &AutoAwayRule{}

func (a *AutoAwayRule) Evaluate(update *poller.Update) (TargetState, error) {
	next := TargetState{ZoneID: a.ZoneID, ZoneName: a.ZoneName}

	if err := a.load(update); err != nil {
		return next, err
	}

	home, away := a.getDeviceStates(update)
	allAway := len(home) == 0 && len(away) > 0
	someoneHome := len(home) > 0
	currentState := poller.GetZoneState(update.ZoneInfo[a.ZoneID])
	overlay := update.ZoneInfo[a.ZoneID].Overlay

	if allAway {
		if currentState != poller.ZoneStateOff {
			next.Action = true
			next.State = poller.ZoneStateOff
			next.Delay = a.Delay
			next.Reason = makeReason(away, "away")
		} else {
			next.Reason = makeReason(away, "away")
		}
	} else if someoneHome {
		if currentState != poller.ZoneStateAuto && overlay.Setting.Power == "OFF" {
			next.Action = true
			next.State = poller.ZoneStateAuto
			next.Delay = 0
			next.Reason = makeReason(home, "home")
		} else {
			next.Reason = makeReason(home, "home")
		}
	}
	return next, nil
}

func (a *AutoAwayRule) getDeviceStates(update *poller.Update) ([]string, []string) {
	var home, away []string
	for _, id := range a.MobileDeviceIDs {
		if entry, exists := update.UserInfo[id]; exists {
			switch entry.IsHome() {
			case tado.DeviceAway:
				away = append(away, entry.Name)
			case tado.DeviceHome:
				home = append(home, entry.Name)
			}
		}
	}
	return home, away
}

func makeReason(users []string, state string) string {
	var verb string
	if len(users) == 1 {
		verb = "is"
	} else {
		verb = "are"
	}
	return strings.Join(users, ", ") + " " + verb + " " + state
}

func (a *AutoAwayRule) load(update *poller.Update) error {
	if len(a.MobileDeviceIDs) > 0 {
		return nil
	}

	for _, user := range a.Users {
		if userID, ok := update.GetUserID(user); ok {
			a.MobileDeviceIDs = append(a.MobileDeviceIDs, userID)
		} else {
			return fmt.Errorf("invalid user: %s", user)
		}
	}

	return nil
}
