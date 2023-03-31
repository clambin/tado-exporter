package rules

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
	"testing"
	"time"
)

func TestTargetState_LogValue(t *testing.T) {
	type fields struct {
		ZoneID   int
		ZoneName string
		Action   bool
		State    ZoneState
		Delay    time.Duration
		Reason   string
	}
	tests := []struct {
		name   string
		fields fields
		want   slog.Value
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := TargetState{
				ZoneID:   tt.fields.ZoneID,
				ZoneName: tt.fields.ZoneName,
				Action:   tt.fields.Action,
				State:    tt.fields.State,
				Delay:    tt.fields.Delay,
				Reason:   tt.fields.Reason,
			}
			assert.Equalf(t, tt.want, s.LogValue(), "LogValue()")
		})
	}
}
