package controller

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_coreAction(t *testing.T) {
	type want struct {
		description    string
		descriptionDue string
		logValue       string
	}
	tests := []struct {
		name string
		coreAction
		want
	}{
		{
			name:       "home: auto",
			coreAction: coreAction{homeState{false, false}, "test", 15 * time.Minute},
			want:       want{"AUTO mode", "AUTO mode in 15m0s", "[state=[overlay=false home=false] delay=15m0s reason=test]"},
		},
		{
			name:       "home: auto",
			coreAction: coreAction{homeState{false, true}, "test", 15 * time.Minute},
			want:       want{"AUTO mode", "AUTO mode in 15m0s", "[state=[overlay=false home=true] delay=15m0s reason=test]"},
		},
		{
			name:       "home: manual away",
			coreAction: coreAction{homeState{true, false}, "test", 15 * time.Minute},
			want:       want{"AWAY mode", "AWAY mode in 15m0s", "[state=[overlay=true home=false] delay=15m0s reason=test]"},
		},
		{
			name:       "home: manual home",
			coreAction: coreAction{homeState{true, true}, "test", 15 * time.Minute},
			want:       want{"HOME mode", "HOME mode in 15m0s", "[state=[overlay=true home=true] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: auto mode",
			coreAction: coreAction{zoneState{false, false}, "test", 15 * time.Minute},
			want:       want{"to auto mode", "to auto mode in 15m0s", "[state=[overlay=false heating=false] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: auto mode",
			coreAction: coreAction{zoneState{false, true}, "test", 15 * time.Minute},
			want:       want{"to auto mode", "to auto mode in 15m0s", "[state=[overlay=false heating=true] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: heating off",
			coreAction: coreAction{zoneState{true, false}, "test", 15 * time.Minute},
			want:       want{"off", "off in 15m0s", "[state=[overlay=true heating=false] delay=15m0s reason=test]"},
		},
		{
			name:       "zone: heating on",
			coreAction: coreAction{zoneState{true, true}, "test", 15 * time.Minute},
			want:       want{"on", "on in 15m0s", "[state=[overlay=true heating=true] delay=15m0s reason=test]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.description, tt.coreAction.Description(false))
			assert.Equal(t, tt.want.descriptionDue, tt.coreAction.Description(true))
			assert.Equal(t, tt.want.logValue, tt.coreAction.LogValue().String())
		})
	}
}
