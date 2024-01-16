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

func (r AutoAwayRule) Evaluate(update poller.Update) (action.Action, error) {
	a := action.Action{
		Label: r.zoneName,
		State: &State{
			zoneID:   r.zoneID,
			zoneName: r.zoneName,
			mode:     action.NoAction,
		},
	}

	if !update.Home {
		a.Reason = "home in AWAY mode"
		return a, nil
	}

	home, away := update.GetDeviceStatus(r.mobileDeviceIDs...)
	allAway := len(home) == 0 && len(away) > 0
	someoneHome := len(home) > 0
	currentState := tadotools.GetZoneState(update.ZoneInfo[r.zoneID])

	if allAway {
		a.Reason = r.makeReason(away, "away")
		if currentState.Heating() {
			a.Delay = r.delay
			a.State.(*State).mode = action.ZoneInOverlayMode
		}
	} else if someoneHome {
		a.Reason = r.makeReason(home, "home")
		if !currentState.Heating() && currentState.Overlay == tado.PermanentOverlay {
			a.State.(*State).mode = action.ZoneInAutoMode
		}
	}

	r.logger.Debug("evaluated",
		slog.Bool("home", bool(update.Home)),
		slog.Any("devices", update.UserInfo),
		slog.Any("result", a),
	)

	return a, nil
}

func (r AutoAwayRule) makeReason(users []string, state string) string {
	var verb string
	if len(users) == 1 {
		verb = "is"
	} else {
		verb = "are"
	}
	return strings.Join(users, ", ") + " " + verb + " " + state
}
