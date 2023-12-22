package rules_test

import (
	"bytes"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"log/slog"
	"testing"
	"time"
)

func TestKind_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		want    rules.Kind
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "autoAway", want: rules.AutoAway, wantErr: assert.NoError},
		{name: "limitOverlay", want: rules.LimitOverlay, wantErr: assert.NoError},
		{name: "nightTime", want: rules.NightTime, wantErr: assert.NoError},
		{name: "invalid", wantErr: assert.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output rules.Kind
			tt.wantErr(t, yaml.Unmarshal([]byte(tt.name), &output))
			assert.Equal(t, tt.want, output)
		})
	}
}

func TestKind_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   rules.Kind
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "autoAway", input: rules.AutoAway, want: "autoAway\n", wantErr: assert.NoError},
		{name: "limitOverlay", input: rules.LimitOverlay, want: "limitOverlay\n", wantErr: assert.NoError},
		{name: "nightTime", input: rules.NightTime, want: "nightTime\n", wantErr: assert.NoError},
		{name: "bad", input: rules.Kind(-1), wantErr: assert.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := yaml.Marshal(tt.input)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, string(output))

			output, err = yaml.Marshal(&tt.input)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, string(output))
		})
	}
}

func TestLoad(t *testing.T) {
	testCases := []struct {
		name    string
		config  string
		wantErr assert.ErrorAssertionFunc
		want    []rules.RuleConfig
	}{
		{
			name: "limitOverlay",
			config: `
zones:
  - zone: "Bathroom"
    rules:
      - kind: limitOverlay
        delay: 1h
`,
			wantErr: assert.NoError,
			want: []rules.RuleConfig{
				{Kind: rules.LimitOverlay, Delay: time.Hour},
			},
		},
		{
			name: "combined",
			config: `
zones:
  - zone: "Bathroom"
    rules:
      - kind: limitOverlay
        delay: 1h
      - kind: autoAway
        delay: 30m
        users: [ foo ]
`,
			wantErr: assert.NoError,
			want: []rules.RuleConfig{
				{Kind: rules.LimitOverlay, Delay: time.Hour},
				{Kind: rules.AutoAway, Delay: 30 * time.Minute, Users: []string{"foo"}},
			},
		},
		{
			name:    "invalid",
			config:  `invalid yaml`,
			wantErr: assert.Error,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := rules.Load(bytes.NewBufferString(tt.config), slog.Default())
			tt.wantErr(t, err)
			if err != nil {
				return
			}
			require.Len(t, cfg, 1)
			assert.Equal(t, tt.want, cfg[0].Rules)
		})
	}
}
