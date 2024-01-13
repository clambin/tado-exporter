package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"github.com/pkg/errors"
	"log/slog"
	"strings"
	"time"
)

type AutoAwayRule struct {
	zoneID          int
	zoneName        string
	delay           time.Duration
	mobileDeviceIDs []int
	logger          *slog.Logger
}

func LoadAutoAwayRule(id int, name string, cfg configuration.AutoAwayConfiguration, update poller.Update, logger *slog.Logger) (AutoAwayRule, error) {
	var deviceIDs []int
	for _, user := range cfg.Users {
		deviceID, ok := update.GetUserID(user)
		if !ok {
			return AutoAwayRule{}, errors.New("invalid mobile name: " + user)
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	return AutoAwayRule{
		zoneID:          id,
		zoneName:        name,
		delay:           cfg.Delay,
		mobileDeviceIDs: deviceIDs,
		logger:          logger.With(slog.String("rule", "autoAway")),
	}, nil
}

var _ rules.Evaluator = AutoAwayRule{}

func (a AutoAwayRule) Evaluate(update poller.Update) (action.Action, error) {
	e := action.Action{
		Label: a.zoneName,
		State: &State{
			zoneID:   a.zoneID,
			zoneName: a.zoneName,
			mode:     action.NoAction,
		},
	}

	if !update.Home {
		e.Reason = "home in AWAY mode"
		return e, nil
	}

	home, away := a.getDeviceStates(update)
	allAway := len(home) == 0 && len(away) > 0
	someoneHome := len(home) > 0
	currentState := tadotools.GetZoneState(update.ZoneInfo[a.zoneID])

	if allAway {
		e.Reason = a.makeReason(away, "away")
		if currentState.Heating() {
			e.Delay = a.delay
			e.State.(*State).mode = action.ZoneInOverlayMode
		}
	} else if someoneHome {
		e.Reason = a.makeReason(home, "home")
		if !currentState.Heating() && currentState.Overlay == tado.PermanentOverlay {
			e.State.(*State).mode = action.ZoneInAutoMode
		}
	}

	a.logger.Debug("evaluated",
		slog.Bool("home", bool(update.Home)),
		slog.Any("result", e),
	)

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

func (a AutoAwayRule) makeReason(users []string, state string) string {
	var verb string
	if len(users) == 1 {
		verb = "is"
	} else {
		verb = "are"
	}
	return strings.Join(users, ", ") + " " + verb + " " + state
}
