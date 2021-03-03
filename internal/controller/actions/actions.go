package actions

import (
	"fmt"
	"github.com/clambin/tado-exporter/pkg/tado"
)

// Actions implements a set of Tado actions controller rules may need
type Actions struct {
	tado.API
}

// Action structure for one Tado action
type Action struct {
	Overlay           bool
	ZoneID            int
	TargetTemperature float64
}

func (action *Action) String() string {
	if action.Overlay {
		return fmt.Sprintf("setOverlay{zoneID=%d, temp=%.1f}", action.ZoneID, action.TargetTemperature)
	} else {
		return fmt.Sprintf("deleteOverlay{zoneID=%d}", action.ZoneID)
	}
}

// RunAction execute an action required by autoAway/overlayLimit rules
//
// We split this out for two reasons: 1. it makes the code more readable
// and 2. makes it easier to write unit tests
//
// Currently supports creating & deleting a fixed overlay
func (actions *Actions) RunAction(action Action) (err error) {
	if action.Overlay == false {
		err = actions.DeleteZoneOverlay(action.ZoneID)
	} else {
		// Set the overlay
		err = actions.SetZoneOverlay(action.ZoneID, action.TargetTemperature)
	}
	return
}

// RunActions calls RunAction for the specified list of Actions
func (actions *Actions) RunActions(actionList []Action) (err error) {
	for _, action := range actionList {
		if err2 := actions.RunAction(action); err2 != nil {
			err = err2
		}
	}
	return
}
