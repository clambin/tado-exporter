package notifier

import (
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/slack-go/slack"
	"log/slog"
	"sync"
)

type SlackNotifier struct {
	Logger slog.Logger
	SlackSender
	channels []slack.Channel
	lock     sync.Mutex
}

type SlackSender interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	GetConversations(params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
}

var _ Notifier = &SlackNotifier{}

func (s *SlackNotifier) Notify(action ScheduleType, state action.Action) {
	channels, err := s.getChannels()
	if err != nil {
		s.Logger.Error("notifier failed to retrieve channels", "err", err)
		return
	}
	for _, channel := range channels {
		if channel.IsMember {
			_, _, err = s.SlackSender.PostMessage(channel.ID, slack.MsgOptionAttachments(slack.Attachment{
				Color: "good",
				Title: buildMessage(action, state),
				Text:  state.Reason,
			}))
		}
	}
}

func (s *SlackNotifier) getChannels() ([]slack.Channel, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.channels == nil {
		var cursor string
		for {
			channels, nextCursor, err := s.SlackSender.GetConversations(&slack.GetConversationsParameters{Cursor: cursor, Limit: 100})
			if err != nil {
				return nil, err
			}
			s.channels = append(s.channels, channels...)
			cursor = nextCursor
			if cursor == "" {
				break
			}
		}
	}
	return s.channels, nil
}
