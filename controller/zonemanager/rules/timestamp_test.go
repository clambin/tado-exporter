package rules_test

import (
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestTimestamp_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		pass  bool
		want  rules.Timestamp
	}{
		{
			name:  "long",
			input: "23:30:15",
			pass:  true,
			want:  rules.Timestamp{Hour: 23, Minutes: 30, Seconds: 15},
		},
		{
			name:  "short",
			input: "23:30",
			pass:  true,
			want:  rules.Timestamp{Hour: 23, Minutes: 30, Seconds: 0},
		},
		{
			name:  "invalid",
			input: "aa:30:00",
			pass:  false,
		},
		{
			name:  "too long",
			input: "123:30:00",
			pass:  false,
		},
		{
			name:  "too short",
			input: "23",
			pass:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output rules.Timestamp
			err := yaml.Unmarshal([]byte(tt.input), &output)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
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
