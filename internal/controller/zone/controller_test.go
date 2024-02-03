package zone_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/controller/zone"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestZoneController(t *testing.T) {
	cfg := configuration.ZoneConfiguration{
		Name: "room",
		Rules: configuration.ZoneRuleConfiguration{
			LimitOverlay: configuration.LimitOverlayConfiguration{
				Delay: time.Hour,
			}},
	}

	z := zone.New(nil, nil, nil, cfg, slog.Default())

	tests := []struct {
		name   string
		update poller.Update
		action string
		delay  time.Duration
	}{
		{
			name: "no overlay",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				Home:     true,
			},
			action: "no action",
			delay:  0,
		},
		{
			name: "permanent overlay (home)",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay())},
				Home:     true,
			},
			action: "moving to auto mode",
			delay:  time.Hour,
		},
		{
			name: "permanent overlay (away)",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay())},
				Home:     false,
			},
			action: "moving to auto mode",
			delay:  0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := z.Evaluate(tt.update)
			assert.NoError(t, err)
			assert.Equal(t, tt.action, a.String())
			assert.Equal(t, tt.delay, a.Delay)
		})
	}
}
