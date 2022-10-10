package zonemanager

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/zonemanager/rules"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/slack-go/slack"
	"time"
)

type Poster struct {
	slackbot.SlackBot
}

func (p *Poster) NotifyQueued(state *rules.NextState) {
	if p.SlackBot != nil {
		p.SlackBot.GetPostChannel() <- []slack.Attachment{{
			Color: "good",
			Title: state.ZoneName + ": " + state.ActionReason,
			Text:  getAction(state) + " in " + state.Delay.Round(time.Second).String(),
		}}
	}
}

func (p *Poster) NotifyCanceled(state *rules.NextState) {
	if p.SlackBot != nil {
		p.SlackBot.GetPostChannel() <- []slack.Attachment{{
			Color: "good",
			Title: state.ZoneName + ": " + state.CancelReason,
			Text:  "cancel " + getAction(state),
		}}
	}
}

func (p *Poster) NotifyAction(state *rules.NextState) {
	if p.SlackBot != nil {
		p.SlackBot.GetPostChannel() <- []slack.Attachment{{
			Color: "good",
			Title: state.ZoneName + ": " + state.ActionReason,
			Text:  getAction(state),
		}}
	}
}

func getAction(state *rules.NextState) (text string) {
	switch state.State {
	case tado.ZoneStateAuto:
		text = "moving to auto mode"
	case tado.ZoneStateOff:
		text = "switching off heating"
	}

	return
}
