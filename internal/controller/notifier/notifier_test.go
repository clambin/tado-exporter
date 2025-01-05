package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/notifier/mocks"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"log/slog"
	"testing"
)

func TestNotifiers_Notify(t *testing.T) {
	b := mocks.NewSlackSender(t)
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := Notifiers{
		&SLogNotifier{Logger: l},
		&SlackNotifier{SlackSender: b, Logger: l},
	}
	channels := []slack.Channel{
		{IsMember: true, GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "1"}}},
		{IsMember: false, GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "2"}}},
	}

	b.EXPECT().
		AuthTest().
		Return(&slack.AuthTestResponse{UserID: "U123456789G"}, nil)
	b.EXPECT().
		GetConversations(mock.AnythingOfType("*slack.GetConversationsParameters")).
		Return(channels, "", nil)
	b.EXPECT().
		PostMessage("1", mock.Anything).
		RunAndReturn(func(channel string, options ...slack.MsgOption) (string, string, error) {
			assert.Equal(t, channel, channels[0].ID)
			return "", "", nil
		}).
		Once()

	n.Notify("foo")
}
