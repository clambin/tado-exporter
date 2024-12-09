package luart

import (
	"github.com/Shopify/go-lua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_tado_isInRange(t *testing.T) {
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
			name: "not in range - before",
			args: args{
				startHour: 22, startMin: 30,
				endHour: 01, endMin: 30,
				currentTime: time.Date(2024, time.November, 26, 22, 0, 0, 0, time.Local),
			},
			want: assert.False,
		},
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
			name: "not in range - after",
			args: args{
				startHour: 22, startMin: 30,
				endHour: 23, endMin: 30,
				currentTime: time.Date(2024, time.November, 26, 23, 45, 0, 0, time.Local),
			},
			want: assert.False,
		},
		{
			name: "multi day, in range, before midnight",
			args: args{
				startHour: 23, startMin: 00,
				endHour: 06, endMin: 00,
				currentTime: time.Date(2024, time.November, 26, 23, 30, 0, 0, time.Local),
			},
			want: assert.True,
		},
		{
			name: "multi day, in range, after midnight",
			args: args{
				startHour: 23, startMin: 00,
				endHour: 06, endMin: 00,
				currentTime: time.Date(2024, time.November, 26, 1, 00, 0, 0, time.Local),
			},
			want: assert.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewWithTime(func() time.Time { return tt.args.currentTime })
			script := `
function validate(startHour, startMin, endHour, endMin)
	return tado.isInRange(startHour, startMin, endHour, endMin)
end
			   `
			require.NoError(t, lua.DoString(l, script))

			l.Global("validate")
			assert.False(t, l.IsNil(-1))
			l.PushNumber(float64(tt.args.startHour))
			l.PushNumber(float64(tt.args.startMin))
			l.PushNumber(float64(tt.args.endHour))
			l.PushNumber(float64(tt.args.endMin))

			err := l.ProtectedCall(4, 1, 0)
			require.NoError(t, err)

			resp := l.ToBoolean(-1)
			tt.want(t, resp)
		})
	}
}

func Test_tado_secondsTillWithNow(t *testing.T) {
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
			l := NewWithTime(func() time.Time { return tt.args.currentTime })

			require.NoError(t, lua.DoString(l, `
function validate(hh, mm)
	return tado.secondsTill(hh, mm)
end
`))

			l.Global("validate")
			require.False(t, l.IsNil(-1))
			l.PushNumber(float64(tt.args.toHour))
			l.PushNumber(float64(tt.args.toMin))

			require.NoError(t, l.ProtectedCall(2, 1, 0))

			delta, ok := l.ToInteger(-1)
			require.True(t, ok)
			assert.Equal(t, tt.want, delta)
		})
	}
}
