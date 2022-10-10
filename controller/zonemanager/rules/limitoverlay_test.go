package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLimitOverlayRule_Evaluate(t *testing.T) {
	var testCases = []testCase{
		{
			name:   "auto mode",
			update: &poller.Update{ZoneInfo: map[int]tado.ZoneInfo{10: {}}},
			action: nil,
		},
		{
			name: "manual control",
			update: &poller.Update{ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}}}},
			action: &NextState{ZoneID: 10, ZoneName: "living room", State: tado.ZoneStateAuto, Delay: time.Hour, ActionReason: "manual temp setting detected", CancelReason: "room no longer in manual temp setting"},
		},
		{
			name: "manual control w/ expiration",
			update: &poller.Update{ZoneInfo: map[int]tado.ZoneInfo{10: {Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 18.0}},
				Termination: tado.ZoneInfoOverlayTermination{Type: "AUTO", RemainingTime: 300},
			}}}},
			action: nil,
		},
	}
	r := &LimitOverlayRule{
		zoneID:   10,
		zoneName: "living room",
		config: &configuration.ZoneLimitOverlay{
			Enabled: true,
			Delay:   time.Hour,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := r.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}
