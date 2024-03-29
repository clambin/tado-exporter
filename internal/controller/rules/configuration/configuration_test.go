package configuration_test

import (
	"bytes"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	c, err := configuration.Load(bytes.NewBufferString(`
home:
  autoAway:
    users: [ "A", "B" ]
    delay: 5m
zones:
  - name: "room"
    rules:
      autoAway:
        users: [ "A" ]
        delay: 1h
      limitOverlay:
        delay: 1h
      nightTime:
        time: 00:00
`))
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, yaml.NewEncoder(&out).Encode(c))
	assert.Equal(t, `home:
    autoAway:
        users:
            - A
            - B
        delay: 5m0s
zones:
    - name: room
      rules:
        autoAway:
            users:
                - A
            delay: 1h0m0s
        limitOverlay:
            delay: 1h0m0s
        nightTime:
            time: "00:00:00"
`, out.String())
}

func Test_IsActive(t *testing.T) {
	type ruleConfig interface {
		IsActive() bool
	}

	testCases := []struct {
		name       string
		cfg        ruleConfig
		wantActive assert.BoolAssertionFunc
	}{
		{
			name:       "autoAway - inactive",
			cfg:        configuration.AutoAwayConfiguration{},
			wantActive: assert.False,
		},
		{
			name:       "autoAway - active",
			cfg:        configuration.AutoAwayConfiguration{Users: []string{"A"}},
			wantActive: assert.True,
		},
		{
			name:       "limitOverlay - inactive",
			cfg:        configuration.LimitOverlayConfiguration{},
			wantActive: assert.False,
		},
		{
			name:       "limitOverlay - active",
			cfg:        configuration.LimitOverlayConfiguration{Delay: time.Hour},
			wantActive: assert.True,
		},
		{
			name:       "nightTime - inactive",
			cfg:        configuration.NightTimeConfiguration{},
			wantActive: assert.False,
		},
		{
			name:       "nightTime - active",
			cfg:        configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{Active: true}},
			wantActive: assert.True,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.wantActive(t, tt.cfg.IsActive())
		})
	}
}

func TestZoneRuleConfiguration_IsActive(t *testing.T) {
	type fields struct {
		AutoAway     configuration.AutoAwayConfiguration
		LimitOverlay configuration.LimitOverlayConfiguration
		NightTime    configuration.NightTimeConfiguration
	}
	tests := []struct {
		name   string
		fields fields
		want   assert.BoolAssertionFunc
	}{
		{
			name: "none",
			want: assert.False,
		},
		{
			name:   "autoAway",
			fields: fields{AutoAway: configuration.AutoAwayConfiguration{Users: []string{"A"}}},
			want:   assert.True,
		},
		{
			name:   "limitOverlay",
			fields: fields{LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour}},
			want:   assert.True,
		},
		{
			name: "nightTime",
			fields: fields{NightTime: configuration.NightTimeConfiguration{Timestamp: configuration.Timestamp{
				Hour:    0,
				Minutes: 0,
				Seconds: 0,
				Active:  true,
			}}},
			want: assert.True,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := configuration.ZoneRuleConfiguration{
				AutoAway:     tt.fields.AutoAway,
				LimitOverlay: tt.fields.LimitOverlay,
				NightTime:    tt.fields.NightTime,
			}
			tt.want(t, c.IsActive())
		})
	}
}
