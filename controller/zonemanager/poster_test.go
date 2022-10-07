package zonemanager

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/slackbot"
	"github.com/clambin/tado-exporter/slackbot/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPoster(t *testing.T) {
	ch := make(slackbot.PostChannel, 1)

	b := mocks.SlackBot{}
	b.On("GetPostChannel").Return(ch)

	p := Poster{SlackBot: &b}

	for _, tt := range []struct {
		state  NextState
		action string
	}{
		{
			state: NextState{
				ZoneID:       1,
				ZoneName:     "foo",
				State:        tado.ZoneStateOff,
				Delay:        time.Hour,
				ActionReason: "bar is away",
				CancelReason: "bar is home",
			},
			action: "switching off heating",
		},
		{
			state: NextState{
				ZoneID:       1,
				ZoneName:     "foo",
				State:        tado.ZoneStateAuto,
				Delay:        0,
				ActionReason: "bar is home",
				CancelReason: "bar is away",
			},
			action: "moving to auto mode",
		},
	} {
		p.NotifyQueued(tt.state)
		a := <-ch
		require.Len(t, a, 1)
		assert.Equal(t, "good", a[0].Color)
		assert.Equal(t, tt.state.ZoneName+": "+tt.state.ActionReason, a[0].Title)
		assert.Equal(t, tt.action+" in "+tt.state.Delay.String(), a[0].Text)

		p.NotifyAction(tt.state)
		a = <-ch
		require.Len(t, a, 1)
		assert.Equal(t, "good", a[0].Color)
		assert.Equal(t, tt.state.ZoneName+": "+tt.state.ActionReason, a[0].Title)
		assert.Equal(t, tt.action, a[0].Text)

		p.NotifyCanceled(tt.state)
		a = <-ch
		require.Len(t, a, 1)
		assert.Equal(t, "good", a[0].Color)
		assert.Equal(t, tt.state.ZoneName+": "+tt.state.CancelReason, a[0].Title)
		assert.Equal(t, "cancel "+tt.action, a[0].Text)
	}
}
