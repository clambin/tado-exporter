package rules

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"log/slog"
	"strconv"
)

type TadoSetter interface {
}

var _ action.State = &State{}

type State struct {
	zoneID          int
	zoneName        string
	mode            action.Mode
	zoneTemperature float64
}

func (s State) Mode() action.Mode {
	return s.mode
}

func (s State) IsEqual(i action.State) bool {
	return s.mode == i.Mode()
}

func (s State) String() string {
	switch s.mode {
	case action.ZoneInOverlayMode:
		if s.zoneTemperature <= 5.0 {
			return "switching off heating"
		}
		return "heating to " + strconv.FormatFloat(s.zoneTemperature, 'f', 1, 64) + "º"
	case action.ZoneInAutoMode:
		return "moving to auto mode"
	default:
		return "no action"
	}
}

func (s State) LogValue() slog.Value {
	values := make([]slog.Attr, 3, 4)

	values[0] = slog.String("type", "zone")
	values[1] = slog.String("name", s.zoneName)
	values[2] = slog.String("mode", s.mode.String())

	if s.mode == action.ZoneInOverlayMode {
		values = append(values, slog.Float64("temperature", s.zoneTemperature))
	}

	return slog.GroupValue(values...)
}

func (s State) Do(ctx context.Context, setter action.TadoSetter) error {
	switch s.mode {
	case action.ZoneInOverlayMode:
		return setter.SetZoneOverlay(ctx, s.zoneID, s.zoneTemperature)
	case action.ZoneInAutoMode:
		return setter.DeleteZoneOverlay(ctx, s.zoneID)
	default:
		return fmt.Errorf("invalid mode: %d", s.mode)
	}
}
