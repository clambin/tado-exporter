package rules_test

import (
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestTimestamp_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    rules.Timestamp
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "long",
			input:   "23:30:15",
			want:    rules.Timestamp{Hour: 23, Minutes: 30, Seconds: 15},
			wantErr: assert.NoError,
		},
		{
			name:    "short",
			input:   "23:30",
			want:    rules.Timestamp{Hour: 23, Minutes: 30, Seconds: 0},
			wantErr: assert.NoError,
		},
		{
			name:    "invalid",
			input:   "aa:30:00",
			wantErr: assert.Error,
		},
		{
			name:    "too long",
			input:   "123:30:00",
			wantErr: assert.Error,
		},
		{
			name:    "too short",
			input:   "23",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output rules.Timestamp
			tt.wantErr(t, yaml.Unmarshal([]byte(tt.input), &output))
			assert.Equal(t, tt.want, output)
		})
	}
}

func TestTimestamp_MarshalYAML(t *testing.T) {
	ts := rules.Timestamp{
		Hour:    23,
		Minutes: 30,
		Seconds: 0,
	}
	output, err := yaml.Marshal(ts)
	require.NoError(t, err)
	assert.Equal(t, "\"23:30:00\"\n", string(output))

	output, err = yaml.Marshal(&ts)
	require.NoError(t, err)
	assert.Equal(t, "\"23:30:00\"\n", string(output))
}
