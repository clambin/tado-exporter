package luart

import (
	"github.com/Shopify/go-lua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestInRange(t *testing.T) {
	type args struct {
		startHour   int
		startMin    int
		endHour     int
		endMin      int
		currentTime time.Time
	}
	tests := []struct {
		name string
		args args
		want assert.BoolAssertionFunc
	}{
		{
			name: "in range",
			args: args{
				startHour: 22, startMin: 30,
				endHour: 23, endMin: 30,
				currentTime: time.Date(2024, time.November, 26, 23, 0, 0, 0, time.Local),
			},
			want: assert.True,
		},
		{
			name: "in range with overlap",
			args: args{
				startHour: 22, startMin: 30,
				endHour: 01, endMin: 30,
				currentTime: time.Date(2024, time.November, 26, 23, 0, 0, 0, time.Local),
			},
			want: assert.True,
		},
		{
			name: "not in range - before",
			args: args{
				startHour: 22, startMin: 30,
				endHour: 01, endMin: 30,
				currentTime: time.Date(2024, time.November, 26, 22, 0, 0, 0, time.Local),
			},
			want: assert.False,
		},
		{
			name: "not in range - after",
			args: args{
				startHour: 22, startMin: 30,
				endHour: 01, endMin: 30,
				currentTime: time.Date(2024, time.November, 26, 2, 0, 0, 0, time.Local),
			},
			want: assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lua.NewState()
			l.PushNumber(float64(tt.args.startHour))
			l.PushNumber(float64(tt.args.startMin))
			l.PushNumber(float64(tt.args.endHour))
			l.PushNumber(float64(tt.args.endMin))

			n := isInRangeWithNow(func() time.Time { return tt.args.currentTime })(l)
			require.Equal(t, 1, n)

			resp := l.ToBoolean(-1)
			tt.want(t, resp)
		})
	}
}

func Test_secondsTillWithNow(t *testing.T) {
	type args struct {
		toHour      int
		toMin       int
		currentTime time.Time
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "before",
			args: args{
				toHour:      14,
				toMin:       30,
				currentTime: time.Date(2024, time.November, 26, 14, 0, 0, 0, time.Local),
			},
			want: 30 * 60, // 30min
		},
		{
			name: "after",
			args: args{
				toHour:      14,
				toMin:       30,
				currentTime: time.Date(2024, time.November, 26, 15, 0, 0, 0, time.Local),
			},
			want: 23*60*60 + 30*60, // 23h30min
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lua.NewState()
			l.PushNumber(float64(tt.args.toHour))
			l.PushNumber(float64(tt.args.toMin))

			n := secondsTillWithNow(func() time.Time { return tt.args.currentTime })(l)
			require.Equal(t, 1, n)

			delta, ok := l.ToInteger(-1)
			require.True(t, ok)
			assert.Equal(t, tt.want, delta)
		})
	}
}
