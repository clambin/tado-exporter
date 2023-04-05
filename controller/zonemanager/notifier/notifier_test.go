package notifier_test

import (
	"github.com/clambin/tado"
	slackbot "github.com/clambin/tado-exporter/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/controller/zonemanager/notifier"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

func TestLoggers_Log(t *testing.T) {
	var testCases = []struct {
		action notifier.Action
		state  rules.Action
		color  string
		title  string
		text   string
	}{
		{
			action: notifier.Queued,
			state:  rules.Action{ZoneID: 10, ZoneName: "foo", State: rules.ZoneState{Overlay: tado.NoOverlay}, Delay: time.Hour, Reason: "manual temp detected"},
			color:  "good",
			title:  "foo: moving to auto mode in 1h0m0s",
			text:   "manual temp detected",
		},
		{
			action: notifier.Canceled,
			state:  rules.Action{ZoneID: 10, ZoneName: "foo", State: rules.ZoneState{Overlay: tado.NoOverlay}, Delay: time.Hour, Reason: "room is in auto mode"},
			color:  "good",
			title:  "foo: canceling moving to auto mode",
			text:   "room is in auto mode",
		},
		{
			action: notifier.Queued,
			state:  rules.Action{ZoneID: 10, ZoneName: "foo", State: rules.ZoneState{Overlay: tado.PermanentOverlay}, Delay: time.Hour, Reason: "foo is away"},
			color:  "good",
			title:  "foo: switching off heating in 1h0m0s",
			text:   "foo is away",
		},
		{
			action: notifier.Done,
			state:  rules.Action{ZoneID: 10, ZoneName: "foo", State: rules.ZoneState{Overlay: tado.PermanentOverlay}, Delay: time.Hour, Reason: "foo is away"},
			color:  "good",
			title:  "foo: switching off heating",
			text:   "foo is away",
		},
	}

	for _, tt := range testCases {
		t.Run(strconv.Itoa(int(tt.action)), func(t *testing.T) {
			b := slackbot.NewSlackBot(t)
			l := notifier.Notifiers{
				&notifier.SLogNotifier{},
				&notifier.SlackNotifier{Bot: b},
			}

			b.On("Send", "", mock.AnythingOfType("[]slack.Attachment")).Run(func(args mock.Arguments) {
				require.Len(t, args, 2)
				attachments, ok := args[1].([]slack.Attachment)
				require.True(t, ok)
				require.Len(t, attachments, 1)
				assert.Equal(t, tt.color, attachments[0].Color)
				assert.Equal(t, tt.title, attachments[0].Title)
				assert.Equal(t, tt.text, attachments[0].Text)
			}).Return(nil)
			l.Notify(tt.action, tt.state)

		})
	}
}
