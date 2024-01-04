package bot

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/slackbot"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/zone/rules"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/slack-go/slack"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Bot struct {
	Tado       TadoSetter
	slack      SlackBot
	poller     poller.Poller
	controller Controller
	logger     *slog.Logger
	lock       sync.RWMutex
	update     poller.Update
	updated    bool
}

type TadoSetter interface {
	DeleteZoneOverlay(context.Context, int) error
	SetZoneTemporaryOverlay(context.Context, int, float64, time.Duration) error
}

type SlackBot interface {
	Register(name string, command slackbot.CommandFunc)
	Run(ctx context.Context) error
	Send(channel string, attachments []slack.Attachment) error
}

type Controller interface {
	ReportTasks() []string
}

func New(tado TadoSetter, tadoBot SlackBot, p poller.Poller, controller Controller, logger *slog.Logger) *Bot {
	b := Bot{
		Tado:       tado,
		slack:      tadoBot,
		poller:     p,
		controller: controller,
		logger:     logger.With(slog.String("component", "tadobot")),
	}
	tadoBot.Register("rules", b.ReportRules)
	tadoBot.Register("rooms", b.ReportRooms)
	tadoBot.Register("set", b.SetRoom)
	tadoBot.Register("refresh", b.DoRefresh)
	tadoBot.Register("users", b.ReportUsers)

	return &b
}

// Run the controller
func (b *Bot) Run(ctx context.Context) error {
	b.logger.Debug("started")
	defer b.logger.Debug("stopped")

	ch := b.poller.Subscribe()
	defer b.poller.Unsubscribe(ch)
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			b.lock.Lock()
			b.update = update
			b.updated = true
			b.lock.Unlock()
		}
	}
}

func (b *Bot) ReportRules(_ context.Context, _ ...string) []slack.Attachment {
	text := b.controller.ReportTasks()

	var slackText, slackTitle string
	if len(text) > 0 {
		slackTitle = "rules:"
		slackText = strings.Join(text, "\n")
	} else {
		slackTitle = ""
		slackText = "no rules have been triggered"
	}

	result := []slack.Attachment{{
		Color: "good",
		Title: slackTitle,
		Text:  slackText,
	}}

	b.logger.Debug("rules", "title", result[0].Title, "text", result[0].Text)

	return result
}

func (b *Bot) ReportRooms(_ context.Context, _ ...string) []slack.Attachment {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if !b.updated {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "no updates yet. please check back later",
		}}
	}

	text := make([]string, 0)

	for zoneID, zone := range b.update.Zones {
		if zoneInfo, found := b.update.ZoneInfo[zoneID]; found {
			text = append(text, fmt.Sprintf("%s: %.1fºC (%s)",
				zone.Name,
				zoneInfo.SensorDataPoints.InsideTemperature.Celsius,
				rules.GetZoneState(zoneInfo).String(),
			))
		}
	}

	slackColor := "bad"
	slackTitle := ""
	slackText := "no rooms found"

	if len(text) > 0 {
		slackColor = "good"
		slackTitle = "rooms:"
		slices.Sort(text)
		slackText = strings.Join(text, "\n")
	}

	return []slack.Attachment{{
		Color: slackColor,
		Title: slackTitle,
		Text:  slackText,
	}}
}

func (b *Bot) SetRoom(ctx context.Context, args ...string) []slack.Attachment {
	b.lock.RLock()
	defer b.lock.RUnlock()

	zoneID, zoneName, auto, temperature, duration, err := b.parseSetCommand(args...)

	if err != nil {
		err = fmt.Errorf("invalid command: %w", err)
	}

	if err == nil {
		if auto {
			err = b.Tado.DeleteZoneOverlay(ctx, zoneID)
		} else {
			err = b.Tado.SetZoneTemporaryOverlay(ctx, zoneID, temperature, duration)
		}
	}

	if err != nil {
		return []slack.Attachment{{
			Color: "bad",
			Title: "",
			Text:  err.Error(),
		}}
	}

	b.poller.Refresh()

	var text string
	if auto {
		text = "Setting " + zoneName + " to automatic mode"
	} else {
		text = fmt.Sprintf("Setting target temperature for %s to %.1fºC", zoneName, temperature)
		if duration > 0 {
			text += " for " + duration.String()
		}
	}
	return []slack.Attachment{{
		Color: "good",
		Title: "",
		Text:  text,
	}}
}

func (b *Bot) parseSetCommand(args ...string) (zoneID int, zoneName string, auto bool, temperature float64, duration time.Duration, err error) {
	if len(args) < 2 {
		err = fmt.Errorf("missing parameters\nUsage: set <room> [auto|<temperature> [<duration>]")
		return
	}

	zoneName = args[0]

	var found bool
	for id, zone := range b.update.Zones {
		if zone.Name == zoneName {
			zoneID = id
			found = true
			break
		}
	}

	if !found {
		err = fmt.Errorf("invalid room name")
		return
	}

	if args[1] == "auto" {
		auto = true
		return
	}

	temperature, err = strconv.ParseFloat(args[1], 64)

	if err != nil {
		err = fmt.Errorf("invalid target temperature: \"%s\"", args[1])
		return
	}

	if len(args) > 2 {
		duration, err = time.ParseDuration(args[2])

		if err != nil {
			err = fmt.Errorf("invalid duration: \"%s\"", args[2])
		}
	}

	return
}

func (b *Bot) DoRefresh(_ context.Context, _ ...string) []slack.Attachment {
	b.poller.Refresh()
	return []slack.Attachment{{
		Text: "refreshing Tado data",
	}}
}

func (b *Bot) ReportUsers(_ context.Context, _ ...string) []slack.Attachment {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if !b.updated {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "no update yet. please check back later",
		}}
	}

	text := make([]string, 0)

	for _, device := range b.update.UserInfo {
		var stateString string
		switch device.IsHome() {
		case tado.DeviceHome:
			stateString = "home"
		case tado.DeviceAway:
			stateString = "away"
		default:
			stateString = "unknown"
		}
		text = append(text, device.Name+": "+stateString)
	}

	return []slack.Attachment{
		{
			Title: "users:",
			Text:  strings.Join(text, "\n"),
		},
	}
}
