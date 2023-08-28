package zone

import (
	"bytes"
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
)

func Test_zoneLogger_LogValue(t *testing.T) {
	tests := []struct {
		name     string
		zoneInfo tado.ZoneInfo
		want     string
	}{
		{
			name:     "auto mode (on)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(19, 20)),
			want:     `level=INFO msg=zone z.settings.power=ON z.settings.temperature=20`,
		},
		{
			name:     "auto mode (off)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(19, 5)),
			want:     `level=INFO msg=zone z.settings.power=OFF`,
		},
		{
			name:     "manual (on)",
			zoneInfo: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(19, 20), testutil.ZoneInfoPermanentOverlay()),
			want:     `level=INFO msg=zone z.settings.power=ON z.settings.temperature=20 z.overlay.termination.type=MANUAL z.overlay.termination.subtype=MANUAL`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := zoneLogger(tt.zoneInfo)

			out := bytes.NewBufferString("")
			opt := slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				// Remove time from the output for predictable test output.
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			}}
			l := slog.New(slog.NewTextHandler(out, &opt))

			l.Log(context.Background(), slog.LevelInfo, "zone", "z", z)
			assert.Equal(t, tt.want+"\n", out.String())
		})
	}
}
