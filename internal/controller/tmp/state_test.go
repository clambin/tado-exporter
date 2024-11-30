package tmp

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestState_homeState(t *testing.T) {
	type args struct {
		home   bool
		manual bool
	}
	tests := []struct {
		name string
		args
		description string
	}{
		{"AWAY auto", args{false, false}, "setting home to AWAY mode"},
		{"AWAY manual", args{false, true}, "setting home to AWAY mode (manual)"},
		{"HOME auto", args{true, false}, "setting home to HOME mode"},
		{"HOME manual", args{true, true}, "setting home to HOME mode (manual)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lua.NewState()
			want := homeState{tt.args.home, tt.args.manual}

			want.ToLua(l)
			got, err := toHomeState(l, -1)
			require.NoError(t, err)
			assert.True(t, got.Equals(want))
			l.Pop(1)

			gotHome, gotManual := got.GetState()
			assert.Equal(t, tt.args.home, gotHome)
			assert.Equal(t, tt.args.manual, gotManual)

			assert.Equal(t, fmt.Sprintf("[home=%v manual=%v]", tt.args.home, tt.args.manual), got.LogValue().String())
			assert.Equal(t, tt.description, got.Description())

		})
	}
}

// TODO
func TestState_zoneState(t *testing.T) {
	l := lua.NewState()
	for _, heating := range []bool{true, false} {
		for _, manual := range []bool{true, false} {
			want := zoneState{heating, manual}

			want.ToLua(l)
			got, err := toZoneState(l, -1)
			require.NoError(t, err)
			assert.True(t, got.Equals(want))
			l.Pop(1)

			gotHeating, gotManual := got.GetState()
			wantLog := fmt.Sprintf("[heating=%v manual=%v]", gotHeating, gotManual)
			assert.Equal(t, wantLog, got.LogValue().String())
		}
	}
}
