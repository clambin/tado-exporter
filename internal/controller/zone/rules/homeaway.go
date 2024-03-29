package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"log/slog"
)

var _ rules.Evaluator = HomeAwayRule{}

type HomeAwayRule struct {
	zoneID   int
	zoneName string
}

func LoadHomeAwayRule(id int, name string, _ poller.Update, _ *slog.Logger) (HomeAwayRule, error) {
	return HomeAwayRule{
		zoneID:   id,
		zoneName: name,
	}, nil
}

func (r HomeAwayRule) Evaluate(update poller.Update) (action.Action, error) {
	e := action.Action{
		Label: r.zoneName,
		//Reason: "home in HOME mode",
		State: &State{
			zoneID:   r.zoneID,
			zoneName: r.zoneName,
			mode:     action.NoAction,
		},
	}

	if update.Home {
		return e, nil
	}

	// LoadZoneRules has already validated zoneID. no need to check here.
	info := update.ZoneInfo[r.zoneID]
	state := tadotools.GetZoneState(info)

	if state.Overlay == tado.NoOverlay {
		e.Reason = "home in AWAY mode, no manual temp setting detected"
		return e, nil
	}

	e.State.(*State).mode = action.ZoneInAutoMode
	e.Reason = "home in AWAY mode, manual temp setting detected"
	return e, nil
}
