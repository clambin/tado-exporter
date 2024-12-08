package slacktools

import "github.com/slack-go/slack"

type Formatter interface {
	Format() slack.MsgOption
	IsZero() bool
}

var _ Formatter = Attachment{}

type Attachment struct {
	Header string
	Body   []string
}

func (t Attachment) Format() slack.MsgOption {
	return slack.MsgOptionBlocks(t.build())
}

func (t Attachment) build() *slack.SectionBlock {
	lines := make([]*slack.TextBlockObject, len(t.Body))
	for i, line := range t.Body {
		lines[i] = slack.NewTextBlockObject(slack.MarkdownType, line, false, false)
	}
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, "*"+t.Header+"*", false, false),
		lines,
		nil,
	)
}

func (t Attachment) IsZero() bool {
	return t.Header == "" && len(t.Body) == 0
}
