package notifier

import (
	"fmt"
	"github.com/slack-go/slack"
	"log/slog"
	"sync"
)

type SlackNotifier struct {
	Logger      *slog.Logger
	SlackSender SlackSender
	userID      string
	lock        sync.Mutex
}

type SlackSender interface {
	PostMessage(string, ...slack.MsgOption) (string, string, error)
	GetConversations(*slack.GetConversationsParameters) ([]slack.Channel, string, error)
	AuthTest() (*slack.AuthTestResponse, error)
	GetUsersInConversation(params *slack.GetUsersInConversationParameters) ([]string, string, error)
}

var _ Notifier = &SlackNotifier{}

func (s *SlackNotifier) Notify(msg string) {
	channels, err := s.getChannels()
	if err != nil {
		s.Logger.Error("notifier failed to retrieve channels", "err", err)
		return
	}
	for _, channel := range channels {
		if _, _, err = s.SlackSender.PostMessage(channel.ID, slack.MsgOptionText(msg, false)); err != nil {
			s.Logger.Error("notifier failed to post message", "err", err)
		}
	}
}

func (s *SlackNotifier) getChannels() ([]slack.Channel, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.userID == "" {
		authResp, err := s.SlackSender.AuthTest()
		if err != nil {
			return nil, fmt.Errorf("AuthTest: %w", err)
		}
		s.userID = authResp.UserID
	}

	var joinedChannels []slack.Channel
	var cursor string
	for {
		channels, nextCursor, err := s.SlackSender.GetConversations(&slack.GetConversationsParameters{Cursor: cursor, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, channel := range channels {
			if channel.IsMember && !channel.IsArchived {
				joinedChannels = append(joinedChannels, channel)
			}
		}
		if cursor = nextCursor; cursor == "" {
			break
		}
	}
	return joinedChannels, nil
}
