package rules

import (
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"strings"
	"time"
)

type AutoAwayRule struct {
	delay           time.Duration
	mobileDeviceIDs []int
}

var _ rules.Evaluator = AutoAwayRule{}

func LoadAutoAwayRule(cfg configuration.AutoAwayConfiguration, update poller.Update) (AutoAwayRule, error) {
	var deviceIDs []int
	for _, user := range cfg.Users {
		deviceID, ok := update.GetUserID(user)
		if !ok {
			return AutoAwayRule{}, fmt.Errorf("invalid mobile name: %s", user)
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	return AutoAwayRule{
		delay:           cfg.Delay,
		mobileDeviceIDs: deviceIDs,
	}, nil
}

func (a AutoAwayRule) Evaluate(update poller.Update) (action.Action, error) {
	evaluation := action.Action{State: State{mode: action.NoAction}}
	home, away := a.getDeviceStates(update)
	if len(home) == 0 {
		evaluation.Reason = makeReason(away, "away")
		if update.Home {
			evaluation.Delay = a.delay
			evaluation.State = State{mode: action.HomeInAwayMode}
		}
	} else {
		evaluation.Reason = makeReason(home, "home")
		if !update.Home {
			evaluation.State = State{mode: action.HomeInHomeMode}
		}
	}
	return evaluation, nil
}

func (a AutoAwayRule) getDeviceStates(update poller.Update) ([]string, []string) {
	var home, away []string
	for _, id := range a.mobileDeviceIDs {
		if entry, exists := update.UserInfo[id]; exists {
			switch entry.IsHome() {
			case tado.DeviceHome:
				home = append(home, entry.Name)
			case tado.DeviceAway, tado.DeviceUnknown:
				away = append(away, entry.Name)
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
