package rules_test

import (
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestKind_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		pass     bool
		expected rules.Kind
	}{
		{name: "autoAway", pass: true, expected: rules.AutoAway},
		{name: "limitOverlay", pass: true, expected: rules.LimitOverlay},
		{name: "nightTime", pass: true, expected: rules.NightTime},
		{name: "invalid", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output rules.Kind
			err := yaml.Unmarshal([]byte(tt.name), &output)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestKind_MarshalYAML(t *testing.T) {
	tests := []struct {
		name  string
		input rules.Kind
		pass  bool
	}{
		{name: "autoAway", input: rules.AutoAway, pass: true},
		{name: "limitOverlay", input: rules.LimitOverlay, pass: true},
		{name: "nightTime", input: rules.NightTime, pass: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := yaml.Marshal(tt.input)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.name+"\n", string(output))

			output, err = yaml.Marshal(&tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.name+"\n", string(output))
		})
	}
}
