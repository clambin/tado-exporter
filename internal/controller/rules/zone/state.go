package zone

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/pkg/tadotools"
	"github.com/clambin/tado/v2"
	"log/slog"
	"strconv"
)

var _ action.State = &State{}

type State struct {
	homeId          tado.HomeId
	zoneID          tado.ZoneId
	zoneName        string
	mode            action.Mode
	zoneTemperature float32
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
		return "heating to " + strconv.FormatFloat(float64(s.zoneTemperature), 'f', 1, 32) + "º"
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
		values = append(values, slog.Float64("temperature", float64(s.zoneTemperature)))
	}

	return slog.GroupValue(values...)
}

func (s State) Do(ctx context.Context, setter action.TadoClient) error {
	switch s.mode {
	case action.ZoneInOverlayMode:
		return tadotools.SetOverlay(ctx, setter, s.homeId, s.zoneID, s.zoneTemperature, 0)
	case action.ZoneInAutoMode:
		_, err := setter.DeleteZoneOverlayWithResponse(ctx, s.homeId, s.zoneID)
		return err
	default:
		return fmt.Errorf("invalid mode: %d", s.mode)
	}
}
