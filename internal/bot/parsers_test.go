package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_parseSetRoom(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		want       setRoomCommand
		wantErr    assert.ErrorAssertionFunc
		errMessage string
	}{
		{
			name:       "Insufficient arguments",
			text:       "room1",
			wantErr:    assert.Error,
			errMessage: "missing parameters",
		},
		{
			name:    "Auto mode",
			text:    "room1 auto",
			want:    setRoomCommand{zoneName: "room1", mode: "auto"},
			wantErr: assert.NoError,
		},
		{
			name:    "Valid temperature, no duration",
			text:    "room1 22.5",
			want:    setRoomCommand{zoneName: "room1", mode: "22.5", temperature: 22.5},
			wantErr: assert.NoError,
		},
		{
			name:       "Invalid temperature",
			text:       "room1 not-a-number",
			wantErr:    assert.Error,
			errMessage: "invalid target temperature",
		},
		{
			name:    "Valid temperature and duration",
			text:    "room1 22.5 2h",
			want:    setRoomCommand{zoneName: "room1", mode: "22.5", temperature: 22.5, duration: 2 * time.Hour},
			wantErr: assert.NoError,
		},
		{
			name:       "Invalid duration",
			text:       "room1 22.5 invalid-duration",
			wantErr:    assert.Error,
			errMessage: "invalid duration",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSetRoom(tt.text)
			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

func Test_tokenizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "one word",
			input: `do`,
			want:  []string{"do"},
		},
		{
			name:  "multiple words",
			input: `a b c `,
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "single-quoted words",
			input: `a 'b c'`,
			want:  []string{"a", "b c"},
		},
		{
			name:  "double-quoted words",
			input: `a "b c"`,
			want:  []string{"a", "b c"},
		},
		{
			name:  "inverse-quoted words",
			input: `a “b c"“`,
			want:  []string{"a", "b c"},
		},
		{
			name:  "empty",
			input: ``,
			want:  nil,
		},
		{
			name:  "empty quote",
			input: `""`,
			want:  []string{""},
		},
		{
			name:  "mismatched quotes",
			input: `"foo`,
			want:  []string{"foo"},
		},
		{
			name:  "empty mismatched quote",
			input: `"`,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tokenizeText(tt.input))
		})
	}
}
