package rules

import (
	"bytes"
	"context"
	"github.com/clambin/tado"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
	"testing"
	"time"
)

func TestAction_LogValue(t *testing.T) {
	type fields struct {
		Action bool
		State  ZoneState
		Delay  time.Duration
		Reason string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "no action",
			fields: fields{Action: false, State: ZoneState{}, Delay: 0, Reason: "bar"},
			want:   "s.id=10 s.name=foo s.action=false s.reason=bar",
		},
		{
			name:   "action: no overlay",
			fields: fields{Action: true, State: ZoneState{Overlay: tado.NoOverlay}, Delay: time.Hour, Reason: "bar"},
			want:   `s.id=10 s.name=foo s.action=true s.state.overlay="no overlay" s.delay=1h0m0s s.reason=bar`,
		},
		{
			name:   "action: overlay",
			fields: fields{Action: true, State: ZoneState{Overlay: tado.PermanentOverlay, TargetTemperature: tado.Temperature{Celsius: 5.0}}, Delay: time.Hour, Reason: "bar"},
			want:   `s.id=10 s.name=foo s.action=true s.state.overlay="permanent overlay" s.state.heating=false s.delay=1h0m0s s.reason=bar`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Action{
				ZoneID:   10,
				ZoneName: "foo",
				Action:   tt.fields.Action,
				State:    tt.fields.State,
				Delay:    tt.fields.Delay,
				Reason:   tt.fields.Reason,
			}

			out := bytes.NewBufferString("")
			l := slog.New(slog.NewTextHandler(out))
			l.Log(context.Background(), slog.LevelInfo, "state", "s", s)

			assert.Contains(t, out.String(), tt.want)
		})
	}
}
