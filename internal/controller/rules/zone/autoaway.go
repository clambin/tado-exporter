package zone

import (
	"errors"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"log/slog"
	"strings"
	"time"
)

// An AutoAwayRule switches off the heating for a zone when all users associated with the zone are AWAY.
//
// Switching off heating is implemented by setting a permanent overlay with a temperature of 5ºC or less.
type AutoAwayRule struct {
	zoneID          tado.ZoneId
	zoneName        string
	delay           time.Duration
	mobileDeviceIDs []tado.MobileDeviceId
	logger          *slog.Logger
}

func LoadAutoAwayRule(id tado.ZoneId, name string, cfg configuration.AutoAwayConfiguration, update poller.Update, logger *slog.Logger) (AutoAwayRule, error) {
	if len(cfg.Users) == 0 {
		return AutoAwayRule{}, errors.New("no users configured")
	}
	var deviceIDs []tado.MobileDeviceId
	for _, user := range cfg.Users {
		device, ok := update.GetMobileDevice(user)
		if !ok {
			return AutoAwayRule{}, errors.New("invalid mobile name: " + user)
		}
		deviceIDs = append(deviceIDs, *device.Id)
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
			homeId:   *update.HomeBase.Id,
			zoneID:   r.zoneID,
			zoneName: r.zoneName,
			mode:     action.NoAction,
		},
	}

	if !update.Home() {
		a.Reason = "home in AWAY mode"
		return a, nil
	}

	home, away := update.GetDeviceState(r.mobileDeviceIDs...)
	allAway := len(home) == 0 && len(away) > 0
	someoneHome := len(home) > 0

	zone, err := update.GetZone(r.zoneName)
	if err != nil {
		return a, err
	}

	if allAway {
		a.Reason = r.makeReason(away, "away")
		if *zone.Setting.Temperature.Celsius > 5 {
			a.Delay = r.delay
			a.State.(*State).mode = action.ZoneInOverlayMode
		}
	} else if someoneHome {
		a.Reason = r.makeReason(home, "home")
		if zone.GetTargetTemperature() <= 5 && zone.Overlay != nil && *zone.Overlay.Termination.Type == tado.ZoneOverlayTerminationTypeMANUAL {
			// TODO: this resets the thermostat if we switched off the heating because the house was in AWAY mode
			// However, if the user switched off the heating, we will immediately switch the heating back on, which is not what the user wanted.
			a.State.(*State).mode = action.ZoneInAutoMode
		}
	}

	r.logger.Debug("evaluated",
		slog.Any("devices", update.MobileDevices),
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
