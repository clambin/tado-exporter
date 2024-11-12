package home

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"log/slog"
	"strings"
	"time"
)

type AutoAwayRule struct {
	usernames set.Set[string]
	delay     time.Duration
	logger    *slog.Logger
}

var _ rules.Evaluator = AutoAwayRule{}

func LoadAutoAwayRule(cfg configuration.AutoAwayConfiguration, update poller.Update, logger *slog.Logger) (AutoAwayRule, error) {
	usernames := set.New[string]()
	for _, username := range cfg.Users {
		if _, ok := update.GetMobileDevice(username); !ok {
			return AutoAwayRule{}, fmt.Errorf("invalid username: %s", username)
		}
		usernames.Add(username)
	}
	return AutoAwayRule{
		usernames: usernames,
		delay:     cfg.Delay,
		logger:    logger.With("rule", "autoAway"),
	}, nil
}

func (a AutoAwayRule) Evaluate(update poller.Update) (action.Action, error) {
	var users, away int
	homeUsers := make([]string, 0, len(a.usernames))
	awayUsers := make([]string, 0, len(a.usernames))
	for device := range update.MobileDevices.GeoTrackedDevices() {
		if a.usernames.Contains(*device.Name) {
			users++
			if *device.Location.AtHome {
				homeUsers = append(homeUsers, *device.Name)
			} else {
				away++
				awayUsers = append(awayUsers, *device.Name)
			}
		}
	}

	var result action.Action
	if users == away {
		result.Reason = makeReason(awayUsers, "away")
		if *update.HomeState.Presence != tado.AWAY {
			result.Delay = a.delay
			result.State = State{mode: action.HomeInAwayMode, homeId: *update.HomeBase.Id}
		}
	} else {
		result.Reason = makeReason(homeUsers, "home")
		if *update.HomeState.Presence != tado.HOME {
			result.State = State{mode: action.HomeInHomeMode, homeId: *update.HomeBase.Id}
		}
	}

	a.logger.Debug("evaluated",
		slog.String("home", string(*update.Presence)),
		slog.Any("devices", update.MobileDevices),
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
