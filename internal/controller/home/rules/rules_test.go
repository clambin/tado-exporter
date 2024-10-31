package rules

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestLoadHomeRules(t *testing.T) {
	cfg := configuration.HomeConfiguration{
		AutoAway: configuration.AutoAwayConfiguration{Users: []string{"A", "B"}, Delay: 30 * time.Minute},
	}
	update := poller.Update{
		MobileDevices: []tado.MobileDevice{
			{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}},
			{Id: oapi.VarP[tado.MobileDeviceId](101), Name: oapi.VarP("B"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(false)}},
		},
	}

	_, err := LoadHomeRules(cfg, update, slog.Default())
	assert.Error(t, err)

	cfg.AutoAway.Users = []string{"A"}
	r, err := LoadHomeRules(cfg, update, slog.Default())
	require.NoError(t, err)
	assert.Len(t, r, 1)
}
