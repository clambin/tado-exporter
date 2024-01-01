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
        timestamp: 00:00
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
            timestamp: "00:00:00"
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.wantActive(t, tt.cfg.IsActive())
		})
	}
}
