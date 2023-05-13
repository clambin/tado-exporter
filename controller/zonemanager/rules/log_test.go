package rules

import (
	"bytes"
	"context"
	"github.com/clambin/tado"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
	"testing"
)

func Test_zoneInfo_LogValue(t *testing.T) {
	tests := []struct {
		name string
		z    zoneInfo
		want string
	}{
		{
			name: "no overlay (on)",
			z:    zoneInfo{Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 21.0}}},
			want: `level=INFO msg=state s.settings.power=ON s.settings.temperature=21`,
		},
		{
			name: "no overlay (off)",
			z:    zoneInfo{Setting: tado.ZonePowerSetting{Power: "OFF"}},
			want: `level=INFO msg=state s.settings.power=OFF`,
		},
		{
			name: "overlay (on)",
			z: zoneInfo{
				Setting: tado.ZonePowerSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
				Overlay: tado.ZoneInfoOverlay{
					Type: "MANUAL",
					Termination: tado.ZoneInfoOverlayTermination{
						Type:              "MANUAL",
						TypeSkillBasedApp: "MANUAL",
					},
				},
			},
			want: `level=INFO msg=state s.settings.power=ON s.settings.temperature=18 s.overlay.type=MANUAL s.overlay.termination.type=MANUAL s.overlay.termination.subtype=MANUAL`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := bytes.NewBufferString("")
			opt := slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				// Remove time from the output for predictable test output.
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			}}
			l := slog.New(slog.NewTextHandler(out, &opt))

			l.Log(context.Background(), slog.LevelInfo, "state", "s", tt.z)
			assert.Equal(t, tt.want+"\n", out.String())
		})
	}
}
