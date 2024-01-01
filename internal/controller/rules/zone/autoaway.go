package zone

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/rules/evaluate"
	"github.com/clambin/tado-exporter/internal/poller"
	"strings"
	"time"
)

type AutoAwayRule struct {
	zoneID          int
	zoneName        string
	delay           time.Duration
	mobileDeviceIDs []int
}

func LoadAutoAwayRule(id int, name string, cfg configuration.AutoAwayConfiguration, update poller.Update) (AutoAwayRule, error) {
	var deviceIDs []int
	for _, user := range cfg.Users {
		deviceID, ok := update.GetUserID(user)
		if !ok {
			return AutoAwayRule{}, fmt.Errorf("invalid mobile name: %s", user)
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	return AutoAwayRule{
		zoneID:          id,
		zoneName:        name,
		delay:           cfg.Delay,
		mobileDeviceIDs: deviceIDs,
	}, nil
}

var _ evaluate.Evaluator = AutoAwayRule{}

func (a AutoAwayRule) Evaluate(update poller.Update) (evaluate.Evaluation, error) {
	var e evaluate.Evaluation

	home, away := a.getDeviceStates(update)
	allAway := len(home) == 0 && len(away) > 0
	someoneHome := len(home) > 0
	currentState := GetZoneState(update.ZoneInfo[a.zoneID])

	if allAway {
		e.Reason = makeReason(away, "away")
		if currentState.Heating() {
			e.Do = func(ctx context.Context, c evaluate.TadoSetter) error {
				return c.SetZoneOverlay(ctx, a.zoneID, 0)
			}
			e.Delay = a.delay
		}
	} else if someoneHome {
		e.Reason = makeReason(home, "home")
		if !currentState.Heating() && currentState.Overlay == tado.PermanentOverlay {
			e.Do = func(ctx context.Context, c evaluate.TadoSetter) error {
				return c.DeleteZoneOverlay(ctx, a.zoneID)
			}
		}
	}
	return e, nil
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
