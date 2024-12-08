package rules

import (
	"cmp"
	"context"
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"iter"
	"slices"
	"strings"
)

type TadoClient interface {
	SetPresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, body tado.SetPresenceLockJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetPresenceLockResponse, error)
	DeletePresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.DeletePresenceLockResponse, error)
	SetZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, body tado.SetZoneOverlayJSONRequestBody, reqEditors ...tado.RequestEditorFn) (*tado.SetZoneOverlayResponse, error)
	DeleteZoneOverlayWithResponse(ctx context.Context, homeId tado.HomeId, zoneId tado.ZoneId, reqEditors ...tado.RequestEditorFn) (*tado.DeleteZoneOverlayResponse, error)
}

// A Rule determines the next Action, given the current State.
type Rule interface {
	Evaluate(State) (Action, error)
}

// Rules groups a set of rules for a zone or home. The Rules' Evaluate method runs through all contained rules and determines
// the required action, given the current State.
// The GetState function takes a poller.Update and returns the current State.
type Rules struct {
	GetState func(update poller.Update) (State, error)
	rules    []Rule
}

func (r Rules) Count() int {
	return len(r.rules)
}

// Evaluate takes the current update, determines the next Action for each rule and returns the first Action required.
// If no rules require an action, it returns an Action for that current state, with the Reason listing all reasons why an action isn't required.
func (r Rules) Evaluate(currentState State) (Action, error) {
	actions := make([]Action, 0, len(r.rules))
	noActions := make([]Action, 0, len(r.rules))
	for i := range r.rules {
		a, err := r.rules[i].Evaluate(currentState)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule %d: %w", i+1, err)
		}
		if a.IsState(currentState) {
			noActions = append(noActions, a)
		} else {
			actions = append(actions, a)
		}
	}
	if len(actions) > 0 {
		slices.SortFunc(actions, func(a, b Action) int { return cmp.Compare(a.Delay(), b.Delay()) })
		return actions[0], nil
	}
	reasons := set.New[string]()
	for _, a := range noActions {
		reasons.Add(a.Reason())
	}
	noActions[0].setReason(strings.Join(reasons.ListOrdered(), ", "))
	return noActions[0], nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// State is the input argument for Evaluate() functions.  It summarizes the state of a Tadoº installation, i.e.
// the state of the house, its registered mobile devices and its zones.
type State struct {
	Devices
	tado.HomeId
	tado.ZoneId
	HomeState
	ZoneState
}

type HomeState struct {
	Overlay bool
	Home    bool
}

type Devices []Device

func (d Devices) filter(users set.Set[string]) iter.Seq[Device] {
	return func(yield func(Device) bool) {
		for _, device := range d {
			if len(users) == 0 || users.Contains(device.Name) {
				if !yield(device) {
					return
				}
			}
		}
	}
}

type Device struct {
	Name string
	Home bool
}

type ZoneState struct {
	Overlay bool
	Heating bool
}
