package rules_test

import (
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
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
