package rules

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
	"time"
)

func TestRules_LoadZoneRules(t *testing.T) {
	tests := []struct {
		name          string
		config        configuration.ZoneConfiguration
		wantErr       assert.ErrorAssertionFunc
		wantRuleCount int
	}{
		{
			name:    "invalid zone",
			config:  configuration.ZoneConfiguration{Name: "invalid-room"},
			wantErr: assert.Error,
		},
		{
			name:          "no rules",
			config:        configuration.ZoneConfiguration{Name: "room"},
			wantErr:       assert.NoError,
			wantRuleCount: 0,
		},
		{
			name: "autoAway",
			config: configuration.ZoneConfiguration{
				Name: "room",
				Rules: configuration.ZoneRuleConfiguration{
					AutoAway: configuration.AutoAwayConfiguration{Users: []string{"A"}},
				},
			},
			wantErr:       assert.NoError,
			wantRuleCount: 2,
		},
		{
			name: "limitOverlay",
			config: configuration.ZoneConfiguration{
				Name: "room",
				Rules: configuration.ZoneRuleConfiguration{
					AutoAway:     configuration.AutoAwayConfiguration{Users: []string{"A"}},
					LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour},
				},
			},
			wantErr:       assert.NoError,
			wantRuleCount: 3,
		},
		{
			name: "nightTime",
			config: configuration.ZoneConfiguration{
				Name: "room",
				Rules: configuration.ZoneRuleConfiguration{
					AutoAway:     configuration.AutoAwayConfiguration{Users: []string{"A"}},
					LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour},
					NightTime:    configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Hour: 23, Minutes: 30, Seconds: 0, Active: true}},
				},
			},
			wantErr:       assert.NoError,
			wantRuleCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := poller.Update{
				HomeBase:  tado.HomeBase{Id: oapi.VarP[tado.HomeId](1)},
				HomeState: tado.HomeState{Presence: oapi.VarP(tado.HOME)},
				Zones: poller.Zones{
					{
						Zone:      tado.Zone{Id: oapi.VarP(10), Name: oapi.VarP("room")},
						ZoneState: tado.ZoneState{Setting: &tado.ZoneSetting{Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](22.0)}}},
					},
				},
				MobileDevices: []tado.MobileDevice{
					{Id: oapi.VarP[tado.MobileDeviceId](100), Name: oapi.VarP("A"), Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)}, Location: &oapi.LocationHome},
				},
			}

			rules, err := LoadZoneRules(tt.config, u, slog.Default())
			tt.wantErr(t, err)
			assert.Len(t, rules, tt.wantRuleCount)
		})
	}
}

// TODO: tests to validate interaction between different rules
