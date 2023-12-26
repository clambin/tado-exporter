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

func TestGetNextNightTimeDelay(t *testing.T) {
	tests := []struct {
		now      time.Time
		expected time.Duration
	}{
		{
			now:      time.Date(2022, 10, 10, 12, 0, 0, 0, time.Local),
			expected: 11*time.Hour + 30*time.Minute,
		},
		{
			now:      time.Date(2022, 10, 10, 23, 45, 0, 0, time.Local),
			expected: 23*time.Hour + 45*time.Minute,
		},
	}

	limit := Timestamp{
		Hour:    23,
		Minutes: 30,
		Seconds: 0,
	}

	for _, tt := range tests {
		t.Run(tt.now.String(), func(t *testing.T) {
			delay := getNextNightTimeDelay(tt.now, limit)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestNightTimeRule_Evaluate(t *testing.T) {
	testCases := []testCase{
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

	r := &NightTimeRule{
		zoneID:   10,
		zoneName: "living room",
		timestamp: Timestamp{
			Hour:    23,
			Minutes: 30,
			Seconds: 0,
		},
	}

	testForceTime = time.Date(2022, 10, 10, 22, 30, 0, 0, time.Local)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			a, err := r.Evaluate(tt.update)
			require.NoError(t, err)
			assert.Equal(t, tt.action, a)
		})
	}
}
