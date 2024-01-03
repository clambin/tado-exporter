package zone

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestController_Evaluate(t *testing.T) {
	cfg := configuration.ZoneConfiguration{
		Name:  "room",
		Rules: configuration.ZoneRuleConfiguration{LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour}},
	}
	type want struct {
		action      assert.BoolAssertionFunc
		description string
		delay       time.Duration
	}
	testCases := []struct {
		name   string
		update poller.Update
		want   want
	}{
		{
			name: "zone in manual mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo(testutil.ZoneInfoPermanentOverlay())},
				Home:     true,
			},
			want: want{
				action:      assert.True,
				description: "moving to auto mode",
				delay:       time.Hour,
			},
		},
		{
			name: "zone in auto mode",
			update: poller.Update{
				Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
				ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
				Home:     true,
			},
			want: want{
				action:      assert.False,
				description: "no action",
			},
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := New(nil, nil, nil, cfg, slog.Default())

			action, err := c.Processor.Evaluate(tt.update)
			require.NoError(t, err)
			tt.want.action(t, action.IsAction())
			assert.Equal(t, tt.want.description, action.String())
			assert.Equal(t, tt.want.delay, action.Delay)
		})
	}

}
