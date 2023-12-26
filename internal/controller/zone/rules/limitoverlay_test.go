package rules

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLimitOverlayRule_Evaluate(t *testing.T) {
	tests := []testCase{
		{
			name:   "auto mode",
			update: poller.Update{ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()}},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "no manual settings detected"},
		},
		{
			name:   "manual control",
			update: poller.Update{ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoPermanentOverlay())}},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Delay: time.Hour, Reason: "manual temp setting detected"},
		},
		{
			name:   "manual control w/ expiration",
			update: poller.Update{ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoTemperature(18, 18), testutil.ZoneInfoTimerOverlay())}},
			action: Action{ZoneID: 10, ZoneName: "living room", Action: false, Reason: "no manual settings detected"},
		},
	}
	r := &LimitOverlayRule{
		zoneID:   10,
		zoneName: "living room",
		delay:    time.Hour,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := r.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}
