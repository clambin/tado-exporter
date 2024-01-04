package monitor

import (
	"bytes"
	"github.com/clambin/tado-exporter/internal/controller/rules/configuration"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
	"time"
)

func Test_makeTasks(t *testing.T) {
	testCses := []struct {
		name   string
		config string
		rules  string
		length int
	}{
		{
			name: "rules",
			config: `
health:
  addr: :9091
controller:
  tadoBot:
    enabled: true
    token: 1234
`,
			rules: `
zones:
  - name: "Bathroom"
    rules:
      limitOverlay:
        delay: 1h
`,
			length: 8,
		},
		{
			name: "no rules",
			config: `
health:
  addr: :9091
controller:
  tadoBot:
    enabled: true
    token: 1234
`,
			rules:  ``,
			length: 5,
		},
	}

	for _, tt := range testCses {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := viper.New()
			cfg.SetConfigType("yaml")
			config := bytes.NewBufferString(tt.config)
			require.NoError(t, cfg.ReadConfig(config))

			var r configuration.Configuration
			if tt.rules != "" {
				var err error
				r, err = configuration.Load(bytes.NewBufferString(tt.rules))
				require.NoError(t, err)
			}

			tasks := makeTasks(cfg, nil, r, "1.0", prometheus.NewPedanticRegistry(), slog.Default())
			assert.Len(t, tasks, tt.length)
		})
	}
}

func Test_maybeLoadRules(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		wantErr assert.ErrorAssertionFunc
		want    configuration.Configuration
	}{
		{
			name: "valid",
			content: `zones:
  - name: "bathroom"
    rules:
      limitOverlay:
        delay: 1h
`,
			wantErr: assert.NoError,
			want: configuration.Configuration{
				Zones: []configuration.ZoneConfiguration{
					{
						Name: "bathroom",
						Rules: configuration.ZoneRuleConfiguration{
							LimitOverlay: configuration.LimitOverlayConfiguration{Delay: time.Hour},
						},
					},
				},
			},
		},
		{
			name:    "invalid",
			content: `invalid yaml`,
			wantErr: assert.Error,
		},
		{
			name:    "missing",
			content: ``,
			wantErr: assert.NoError,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := os.CreateTemp("", "")
			require.NoError(t, err)

			if tt.content != "" {
				_, err := f.Write([]byte(tt.content))
				require.NoError(t, err)
				_ = f.Close()
				defer func() { _ = os.Remove(f.Name()) }()
			} else {
				_ = f.Close()
				_ = os.Remove(f.Name())
			}

			r, err := maybeLoadRules(f.Name())
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, r)
		})
	}
}
