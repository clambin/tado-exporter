package tadotools

import (
	"fmt"
	"github.com/clambin/tado"
	"time"
)

type ZoneState struct {
	Overlay           tado.OverlayTerminationMode
	Duration          time.Duration
	TargetTemperature tado.Temperature
}

func GetZoneState(zoneInfo tado.ZoneInfo) ZoneState {
	return ZoneState{
		Overlay:           zoneInfo.Overlay.GetMode(),
		Duration:          time.Second * time.Duration(zoneInfo.Overlay.Termination.RemainingTimeInSeconds),
		TargetTemperature: zoneInfo.Setting.Temperature,
	}
}

func (s ZoneState) Heating() bool {
	return s.TargetTemperature.Celsius > 5.0
}

func (s ZoneState) String() any {
	if !s.Heating() {
		return "off"
	}
	switch s.Overlay {
	case tado.NoOverlay:
		return fmt.Sprintf("target: %.1f", s.TargetTemperature.Celsius)
	case tado.PermanentOverlay:
		return fmt.Sprintf("target: %.1f, MANUAL", s.TargetTemperature.Celsius)
	case tado.TimerOverlay, tado.NextBlockOverlay:
		return fmt.Sprintf("target: %.1f, MANUAL for %s", s.TargetTemperature.Celsius, s.Duration)
	default:
		return "unknown"
	}
}

/*
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
*/
