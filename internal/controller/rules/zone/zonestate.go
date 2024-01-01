package zone

import (
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

/*
func (s ZoneState) Action() string {
	if s.Overlay == tado.UnknownOverlay {
		return "unknown action"
	}

	if s.Overlay == tado.NoOverlay {
		return "moving to auto mode"
	}

	var action string
	if !s.Heating() {
		action = "switching off heating"
	} else {
		action = fmt.Sprintf("setting temperature to %.1f", s.TargetTemperature.Celsius)
	}

	if s.Overlay == tado.TimerOverlay || s.Overlay == tado.NextBlockOverlay {
		action += " for " + s.Duration.String()
	}
	return action
}

func (s ZoneState) String() string {
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
	default:
		return fmt.Errorf("unsupported overlay: %s", s.Overlay.String())
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


*/
