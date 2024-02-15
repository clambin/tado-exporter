package home_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/home"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/testutil"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestHomeController(t *testing.T) {
	cfg := configuration.HomeConfiguration{AutoAway: configuration.AutoAwayConfiguration{
		Users: []string{"A"},
		Delay: time.Hour,
	}}

	h := home.New(nil, nil, nil, cfg, slog.Default())

	tests := []struct {
		name   string
		update poller.Update
		action string
		delay  time.Duration
	}{
		{
			name: "home, user home",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(10, "A", testutil.Home(true)),
				},
				Home: true,
			},
			action: "no action",
		},
		{
			name: "home, user away",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(10, "A", testutil.Home(false)),
				},
				Home: true,
			},
			action: "setting home to away mode",
			delay:  time.Hour,
		},
		{
			name: "away, user home",
			update: poller.Update{
				UserInfo: map[int]tado.MobileDevice{
					100: testutil.MakeMobileDevice(10, "A", testutil.Home(true)),
				},
				Home: false,
			},
			action: "setting home to home mode",
			delay:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := h.Evaluate(tt.update)
			assert.NoError(t, err)
			assert.Equal(t, tt.action, a.String())
			assert.Equal(t, tt.delay, a.Delay)
		})
	}
}
