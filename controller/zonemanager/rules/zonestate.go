package rules

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"golang.org/x/exp/slog"
)

type ZoneState struct {
	Overlay           tado.OverlayTerminationMode
	Heating           bool
	TargetTemperature tado.Temperature
}

func (s ZoneState) LogValue() slog.Value {
	values := make([]slog.Attr, 1, 3)
	values[0] = slog.Bool("heating", s.Heating)

	if s.Heating {
		values = append(values, slog.Float64("targer", s.TargetTemperature.Celsius))
	}
	if s.Overlay != tado.NoOverlay {
		values = append(values, slog.String("overlay", s.Overlay.String()))
	}
	return slog.GroupValue(values...)
}

func (s ZoneState) String() string {
	if s.Overlay == tado.NoOverlay {
		return "moving to auto mode"
	}
	if s.Overlay == tado.PermanentOverlay && (!s.Heating || s.TargetTemperature.Celsius <= 5.0) {
		return "switching off heating"
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
		temp := s.TargetTemperature.Celsius
		if temp <= 5.0 || !s.Heating {
			temp = 5.0
		}
		return api.SetZoneOverlay(ctx, zoneID, temp)
	}
	return fmt.Errorf("unsupported overlay: %s", s.Overlay.String())
}

func GetZoneState(zoneInfo tado.ZoneInfo) ZoneState {
	return ZoneState{
		Heating:           zoneInfo.Setting.Power == "ON",
		TargetTemperature: zoneInfo.Setting.Temperature,
		Overlay:           zoneInfo.Overlay.GetMode(),
	}
}
