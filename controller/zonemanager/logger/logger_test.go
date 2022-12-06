package logger_test

import (
	"github.com/clambin/tado"
	slackbot "github.com/clambin/tado-exporter/controller/slackbot/mocks"
	"github.com/clambin/tado-exporter/controller/zonemanager/logger"
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
		action logger.Action
		state  rules.NextState
		color  string
		title  string
		text   string
	}{
		{
			action: logger.Queued,
			state:  rules.NextState{ZoneID: 10, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: time.Hour, ActionReason: "manual temp detected", CancelReason: "room is in auto mode"},
			color:  "good",
			title:  "foo: manual temp detected",
			text:   "moving to auto mode in 1h0m0s",
		},
		{
			action: logger.Done,
			state:  rules.NextState{ZoneID: 10, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: time.Hour, ActionReason: "manual temp detected", CancelReason: "room is in auto mode"},
			color:  "good",
			title:  "foo: manual temp detected",
			text:   "moving to auto mode",
		},
		{
			action: logger.Canceled,
			state:  rules.NextState{ZoneID: 10, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: time.Hour, ActionReason: "manual temp detected", CancelReason: "room is in auto mode"},
			color:  "good",
			title:  "foo: room is in auto mode",
			text:   "cancel moving to auto mode",
		},
		{
			action: logger.Queued,
			state:  rules.NextState{ZoneID: 10, ZoneName: "foo", State: tado.ZoneStateOff, Delay: time.Hour, ActionReason: "foo is away", CancelReason: "foo is home"},
			color:  "good",
			title:  "foo: foo is away",
			text:   "switching off heating in 1h0m0s",
		},
	}

	b := slackbot.NewSlackBot(t)
	l := logger.Loggers{
		&logger.StdOutLogger{},
		&logger.SlackLogger{Bot: b},
	}

	for _, tt := range testCases {
		t.Run(strconv.Itoa(int(tt.action)), func(t *testing.T) {
			b.On("Send", "", mock.AnythingOfType("[]slack.Attachment")).Run(func(args mock.Arguments) {
				require.Len(t, args, 2)
				attachments, ok := args[1].([]slack.Attachment)
				require.True(t, ok)
				require.Len(t, attachments, 1)
				assert.Equal(t, tt.color, attachments[0].Color)
				assert.Equal(t, tt.title, attachments[0].Title)
				assert.Equal(t, tt.text, attachments[0].Text)
			}).Return(nil)
			l.Log(tt.action, &tt.state)

		})
	}
}
