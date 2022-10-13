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

func TestGetNextNightTimeDelay(t *testing.T) {
	type tts struct {
		now      time.Time
		expected time.Duration
	}
	var testCases = []tts{
		{
			now:      time.Date(2022, 10, 10, 12, 0, 0, 0, time.Local),
			expected: 11*time.Hour + 30*time.Minute,
		},
		{
			now:      time.Date(2022, 10, 10, 23, 45, 0, 0, time.Local),
			expected: 23*time.Hour + 45*time.Minute,
		},
	}

	limit := configuration.ZoneNightTimeTimestamp{
		Hour:    23,
		Minutes: 30,
		Seconds: 0,
	}

	for _, tt := range testCases {
		t.Run(tt.now.String(), func(t *testing.T) {
			delay := getNextNightTimeDelay(tt.now, limit)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestNightTimeRule_Evaluate(t *testing.T) {
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

	r := &NightTimeRule{
		zoneID:   10,
		zoneName: "living room",
		config: &configuration.ZoneNightTime{
			Enabled: true,
			Time: configuration.ZoneNightTimeTimestamp{
				Hour:    23,
				Minutes: 30,
				Seconds: 0,
			},
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
