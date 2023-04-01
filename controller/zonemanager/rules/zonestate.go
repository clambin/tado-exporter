package rules

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"golang.org/x/exp/slog"
	"math"
)

type ZoneState struct {
	Overlay           tado.OverlayTerminationMode
	TargetTemperature tado.Temperature
}

func (s ZoneState) Heating() bool {
	return s.TargetTemperature.Celsius > 5.0
}

func (s ZoneState) String() string {
	switch s.Overlay {
	case tado.NoOverlay:
		return "moving to auto mode"
	case tado.PermanentOverlay:
		if !s.Heating() {
			return "switching off heating"
		}
	}
	return "unknown action"
}

//go:generate mockery --name TadoSetter
type TadoSetter interface {
	DeleteZoneOverlay(context.Context, int) error
	SetZoneOverlay(context.Context, int, float64) error
	//SetZoneTemporaryOverlay(context.Context, int, float64, time.Duration) error
}

func (s ZoneState) Do(ctx context.Context, api TadoSetter, zoneID int) error {
	switch s.Overlay {
	case tado.NoOverlay:
		return api.DeleteZoneOverlay(ctx, zoneID)
	case tado.PermanentOverlay:
		return api.SetZoneOverlay(ctx, zoneID, math.Max(s.TargetTemperature.Celsius, 5.0))
	}
	return fmt.Errorf("unsupported overlay: %s", s.Overlay.String())
}

func GetZoneState(zoneInfo tado.ZoneInfo) ZoneState {
	return ZoneState{
		Overlay:           zoneInfo.Overlay.GetMode(),
		TargetTemperature: zoneInfo.Setting.Temperature,
	}
}

func (s ZoneState) LogValue() slog.Value {
	values := make([]slog.Attr, 1, 3)
	values[0] = slog.String("overlay", s.Overlay.String())
	if s.Overlay != tado.NoOverlay {
		values = append(values, slog.Bool("heating", s.Heating()))
		if s.Heating() {
			values = append(values, slog.Float64("target", s.TargetTemperature.Celsius))
		}
	}
	return slog.GroupValue(values...)
}
