package registry

import "fmt"

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
func (registry *Registry) RunAction(action Action) (err error) {
	if action.Overlay == false {
		err = registry.DeleteZoneOverlay(action.ZoneID)
	} else {
		// Set the overlay
		err = registry.SetZoneOverlay(action.ZoneID, action.TargetTemperature)
	}
	return
}

// RunActions calls RunAction for the specified list of Actions
func (registry *Registry) RunActions(actions []Action) (err error) {
	for _, action := range actions {
		if err2 := registry.RunAction(action); err2 != nil {
			err = err2
		}
	}
	return
}
