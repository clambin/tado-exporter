package rules

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

func TestRules_HomeRules(t *testing.T) {
	cfg := configuration.HomeConfiguration{
		AutoAway: configuration.AutoAwayConfiguration{Users: []string{"A", "B"}, Delay: 30 * time.Minute},
	}
	update := poller.Update{
		Zones:    map[int]tado.Zone{10: {ID: 10, Name: "room"}},
		ZoneInfo: map[int]tado.ZoneInfo{10: testutil.MakeZoneInfo()},
		UserInfo: map[int]tado.MobileDevice{100: testutil.MakeMobileDevice(100, "A", testutil.Home(true))},
		Home:     true,
	}

	_, err := LoadHomeRules(cfg, update, slog.Default())
	assert.Error(t, err)

	cfg.AutoAway.Users = []string{"A"}
	r, err := LoadHomeRules(cfg, update, slog.Default())
	require.NoError(t, err)
	assert.Len(t, r, 1)
}
