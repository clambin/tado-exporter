package zone

import (
	"github.com/clambin/tado-exporter/internal/controller/rules"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"log/slog"
)

var _ rules.Evaluator = HomeAwayRule{}

// A HomeAwayRule deletes an overlay from a zone when the home is in AWAY mode
type HomeAwayRule struct {
	zoneID   tado.ZoneId
	zoneName string
}

func LoadHomeAwayRule(id tado.ZoneId, name string, _ poller.Update, _ *slog.Logger) (HomeAwayRule, error) {
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
			homeId:   *update.HomeBase.Id,
			zoneID:   r.zoneID,
			zoneName: r.zoneName,
			mode:     action.NoAction,
		},
	}

	if update.Home() {
		return e, nil
	}

	zone, err := update.GetZone(r.zoneName)
	if err != nil {
		return e, err
	}
	if zone.ZoneState.Overlay == nil {
		e.Reason = "home in AWAY mode, no manual temp setting detected"
		return e, nil
	}

	e.State.(*State).mode = action.ZoneInAutoMode
	e.Reason = "home in AWAY mode, manual temp setting detected"
	return e, nil
}
