package notifier_test

import (
	"github.com/clambin/tado-exporter/internal/controller/notifier"
	"github.com/clambin/tado-exporter/internal/controller/notifier/mocks"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/controller/testutil"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestNotifiers_Notify(t *testing.T) {
	var testCases = []struct {
		name   string
		action notifier.ScheduleType
		state  action.Action
		color  string
		title  string
		text   string
	}{
		{
			name:   "queued",
			action: notifier.Queued,
			state: action.Action{
				State:  testutil.FakeState{ModeValue: action.ZoneInOverlayMode},
				Delay:  time.Hour,
				Reason: "foo",
				Label:  "room",
			},
			color: "good",
			title: "room: overlay in 1h0m0s",
			text:  "foo",
		},
		{
			name:   "canceled",
			action: notifier.Canceled,
			state: action.Action{
				State:  testutil.FakeState{ModeValue: action.ZoneInAutoMode},
				Delay:  time.Hour,
				Reason: "foo",
				Label:  "room",
			},
			color: "good",
			title: "room: canceling auto",
			text:  "foo",
		},
		{
			name:   "done",
			action: notifier.Done,
			state: action.Action{
				State:  testutil.FakeState{ModeValue: action.ZoneInOverlayMode},
				Delay:  time.Hour,
				Reason: "foo",
				Label:  "room",
			},
			color: "good",
			title: "room: overlay",
			text:  "foo",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := mocks.NewSlackSender(t)
			l := notifier.Notifiers{
				&notifier.SLogNotifier{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))},
				&notifier.SlackNotifier{SlackSender: b},
			}

			channels := []slack.Channel{
				{IsMember: true, GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "1"}}},
				{IsMember: false, GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "2"}}},
			}

			b.EXPECT().
				GetConversations(mock.AnythingOfType("*slack.GetConversationsParameters")).
				Return(channels, "", nil).Once()

			b.EXPECT().
				PostMessage("1", mock.Anything).
				RunAndReturn(func(channel string, options ...slack.MsgOption) (string, string, error) {
					assert.Equal(t, channel, channels[0].ID)
					// TODO
					//require.Len(t, attachments, 1)
					//assert.Equal(t, tt.color, attachments[0].Color)
					//assert.Equal(t, tt.title, attachments[0].Title)
					//assert.Equal(t, tt.text, attachments[0].Text)
					return "", "", nil
				}).
				Once()

			l.Notify(tt.action, tt.state)
		})
	}
}
