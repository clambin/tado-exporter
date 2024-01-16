package rules

import (
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"log/slog"
	"strings"
	"time"
)

type AutoAwayRule struct {
	delay           time.Duration
	mobileDeviceIDs []int
	logger          *slog.Logger
}

var _ rules.Evaluator = AutoAwayRule{}

func LoadAutoAwayRule(cfg configuration.AutoAwayConfiguration, update poller.Update, logger *slog.Logger) (AutoAwayRule, error) {
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
		logger:          logger.With("rule", "autoAway"),
	}, nil
}

func (a AutoAwayRule) Evaluate(update poller.Update) (action.Action, error) {
	home, away := update.GetDeviceStatus(a.mobileDeviceIDs...)

	var result action.Action
	if len(home) == 0 {
		result.Reason = makeReason(away, "away")
		if update.Home {
			result.Delay = a.delay
			result.State = State{mode: action.HomeInAwayMode}
		}
	} else {
		result.Reason = makeReason(home, "home")
		if !update.Home {
			result.State = State{mode: action.HomeInHomeMode}
		}
	}

	a.logger.Debug("evaluated",
		slog.Bool("home", bool(update.Home)),
		slog.Any("devices", update.UserInfo),
		slog.Any("result", result),
	)

	return result, nil
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
