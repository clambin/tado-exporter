package zonemanager

import (
	"github.com/clambin/tado-exporter/configuration"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestManager_getNextState(t *testing.T) {
	m := New(nil, nil, nil, config)

	for _, tt := range testCases {
		t.Log(tt.name)
		current, next := m.getNextState(tt.update)
		assert.Equal(t, tt.current, current, tt.name)
		assert.Equal(t, tt.next, next, tt.name)
	}
}

func Test_nightTimeDelay(t *testing.T) {
	type args struct {
		nightTime configuration.ZoneNightTimeTimestamp
		now       time.Time
	}
	tests := []struct {
		name      string
		args      args
		wantDelay time.Duration
	}{
		{
			name: "today",
			args: args{
				nightTime: configuration.ZoneNightTimeTimestamp{Hour: 23, Minutes: 30, Seconds: 0},
				now:       time.Date(2022, 9, 19, 23, 0, 0, 0, time.Local),
			},
			wantDelay: 30 * time.Minute,
		},
		{
			name: "tomorrow",
			args: args{
				nightTime: configuration.ZoneNightTimeTimestamp{Hour: 23, Minutes: 00, Seconds: 0},
				now:       time.Date(2022, 9, 19, 23, 30, 0, 0, time.Local),
			},
			wantDelay: 23*time.Hour + 30*time.Minute,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.wantDelay, nightTimeDelay(tt.args.nightTime, tt.args.now), "nightTimeDelay(%v, %v)", tt.args.nightTime, tt.args.now)
		})
	}
}
