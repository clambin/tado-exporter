package bot

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/tadotools"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Bot struct {
	TadoClient
	SlackApp
	poller.Poller
	Controller
	logger  *slog.Logger
	lock    sync.RWMutex
	update  poller.Update
	updated bool
}

type TadoClient interface {
	action.TadoClient
	DeletePresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.DeletePresenceLockResponse, error)
}

type Controller interface {
	ReportTasks() []string
}

type SlackApp interface {
	AddSlashCommand(string, func(slack.SlashCommand, *socketmode.Client))
	Run(ctx context.Context) error
}

func New(tadoClient TadoClient, app SlackApp, p poller.Poller, c Controller, logger *slog.Logger) *Bot {
	b := Bot{
		TadoClient: tadoClient,
		SlackApp:   app,
		Poller:     p,
		Controller: c,
		logger:     logger,
	}

	b.SlackApp.AddSlashCommand("/rules", b.doAndPost(b.onRules))
	b.SlackApp.AddSlashCommand("/rooms", b.doAndPost(b.onRooms))
	b.SlackApp.AddSlashCommand("/users", b.doAndPost(b.onUsers))
	b.SlackApp.AddSlashCommand("/setroom", b.doAndPost(b.onSetRoom))
	b.SlackApp.AddSlashCommand("/sethome", b.doAndPost(b.onSetHome))
	b.SlackApp.AddSlashCommand("/refresh", b.doAndPost(b.onRefresh))

	return &b
}

// Run the controller
func (b *Bot) Run(ctx context.Context) error {
	b.logger.Debug("slackbot started")
	defer b.logger.Debug("slackbot stopped")
	errCh := make(chan error)
	go func() { errCh <- b.SlackApp.Run(ctx) }()

	ch := b.Poller.Subscribe()
	defer b.Poller.Unsubscribe(ch)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("slackbot failed: %w", err)
			}
		case <-ctx.Done():
			return nil
		case update := <-ch:
			b.setUpdate(update)
		}
	}
}

func (b *Bot) setUpdate(update poller.Update) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.update = update
	b.updated = true
}

func (b *Bot) getUpdate() (poller.Update, bool) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.update, b.updated
}

func (b *Bot) onRules(_ context.Context, _ ...string) slack.Attachment {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.Controller == nil {
		return slack.Attachment{
			Color: "bad",
			Text:  "controller isn't running",
		}
	}

	text := b.Controller.ReportTasks()

	var slackText, slackTitle string
	if len(text) > 0 {
		slackTitle = "rules:"
		slackText = strings.Join(text, "\n")
	} else {
		slackTitle = ""
		slackText = "no rules have been triggered"
	}

	b.logger.Debug("rules", "title", slackTitle, "text", slackText)
	return slack.Attachment{
		Color: "good",
		Title: slackTitle,
		Text:  slackText,
	}
}

func (b *Bot) onRooms(_ context.Context, _ ...string) slack.Attachment {
	update, ok := b.getUpdate()
	if !ok {
		return slack.Attachment{
			Color: "bad",
			Text:  "no updates yet. please check back later",
		}
	}

	text := make([]string, 0, len(update.Zones))

	for _, zone := range update.Zones {
		text = append(text, fmt.Sprintf("%s: %.1fºC (%s)",
			*zone.Name,
			*zone.SensorDataPoints.InsideTemperature.Celsius,
			zoneState(zone),
		))
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

	return slack.Attachment{
		Color: slackColor,
		Title: slackTitle,
		Text:  slackText,
	}
}

func zoneState(zone poller.Zone) string {
	targetTemperature := zone.GetTargetTemperature()
	if targetTemperature < 5.0 {
		return "off"
	}

	if zone.Overlay == nil {
		return fmt.Sprintf("target: %.1f", targetTemperature)
	}
	switch *zone.Overlay.Termination.Type {
	case tado.ZoneOverlayTerminationTypeMANUAL:
		return fmt.Sprintf("target: %.1f, MANUAL", targetTemperature)
	default:
		return fmt.Sprintf("target: %.1f, MANUAL for %s", targetTemperature, (time.Duration(*zone.Overlay.Termination.DurationInSeconds) * time.Second).String())
	}
}

func (b *Bot) onUsers(_ context.Context, _ ...string) slack.Attachment {
	update, updated := b.getUpdate()
	if !updated {
		return slack.Attachment{
			Color: "bad",
			Text:  "no update yet. please check back later",
		}
	}

	text := make([]string, 0)

	for _, device := range update.MobileDevices {
		if !*device.Settings.GeoTrackingEnabled {
			continue
		}
		location := map[bool]string{true: "home", false: "away"}[*device.Location.AtHome]
		text = append(text, *device.Name+": "+location)
	}

	return slack.Attachment{
		Title: "users:",
		Text:  strings.Join(text, "\n"),
	}
}

func (b *Bot) onSetRoom(_ context.Context, args ...string) slack.Attachment {
	update, ok := b.getUpdate()
	if !ok {
		return slack.Attachment{
			Color: "bad",
			Text:  "no updates yet. please check back later",
		}
	}

	zoneID, zoneName, auto, temperature, duration, err := b.parseSetRoomCommand(args...)

	if err != nil {
		err = fmt.Errorf("invalid command: %w", err)
	}

	if err == nil {
		ctx := context.Background()
		if auto {
			_, err = b.TadoClient.DeleteZoneOverlayWithResponse(ctx, *update.HomeBase.Id, zoneID)
		} else {
			err = tadotools.SetOverlay(ctx, b.TadoClient, *update.HomeBase.Id, zoneID, temperature, duration)
		}
	}

	if err != nil {
		return slack.Attachment{
			Color: "bad",
			Title: "",
			Text:  err.Error(),
		}
	}

	b.Poller.Refresh()

	var text string
	if auto {
		text = "Setting " + zoneName + " to automatic mode"
	} else {
		text = fmt.Sprintf("Setting target temperature for %s to %.1fºC", zoneName, temperature)
		if duration > 0 {
			text += " for " + duration.String()
		}
	}
	return slack.Attachment{
		Color: "good",
		Title: "",
		Text:  text,
	}
}

func (b *Bot) parseSetRoomCommand(args ...string) (zoneID int, zoneName string, auto bool, temperature float32, duration time.Duration, err error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if len(args) < 2 {
		err = fmt.Errorf("missing parameters\nUsage: set room <room> [auto|<temperature> [<duration>]")
		return
	}

	zoneName = args[0]

	zone, err := b.update.GetZone(zoneName)
	if err != nil {
		err = fmt.Errorf("invalid room name: %s", zoneName)
		return
	}
	zoneID = *zone.Id

	if args[1] == "auto" {
		auto = true
		return
	}

	temp, err := strconv.ParseFloat(args[1], 32)
	if err != nil {
		err = fmt.Errorf("invalid target temperature: \"%s\"", args[1])
		return
	}
	temperature = float32(temp)

	if len(args) > 2 {
		duration, err = time.ParseDuration(args[2])

		if err != nil {
			err = fmt.Errorf("invalid duration: \"%s\"", args[2])
		}
	}

	return
}

func (b *Bot) onSetHome(ctx context.Context, args ...string) slack.Attachment {
	if len(args) != 1 {
		return slack.Attachment{Color: "bad", Text: "missing parameter\nUsage: set home [home|away|auto]"}
	}

	update, ok := b.getUpdate()
	if !ok {
		return slack.Attachment{
			Color: "bad",
			Text:  "no updates yet. please check back later",
		}
	}

	var err error
	switch args[0] {
	case "home":
		_, err = b.TadoClient.SetPresenceLockWithResponse(ctx, *update.HomeBase.Id, tado.SetPresenceLockJSONRequestBody{HomePresence: oapi.VarP(tado.HOME)})
	case "away":
		_, err = b.TadoClient.SetPresenceLockWithResponse(ctx, *update.HomeBase.Id, tado.SetPresenceLockJSONRequestBody{HomePresence: oapi.VarP(tado.AWAY)})
	case "auto":
		_, err = b.TadoClient.DeletePresenceLockWithResponse(ctx, *update.HomeBase.Id)
	default:
		return slack.Attachment{Color: "bad", Text: "missing parameter\nUsage: set home [home|away|auto]"}
	}

	if err != nil {
		return slack.Attachment{Color: "bad", Text: "failed: " + err.Error()}
	}

	b.Poller.Refresh()

	return slack.Attachment{Color: "good", Text: "set home to " + args[0] + " mode"}
}

func (b *Bot) onRefresh(_ context.Context, _ ...string) slack.Attachment {
	b.Poller.Refresh()
	return slack.Attachment{Text: "refreshing Tado data"}
}

func (b *Bot) doAndPost(f func(context.Context, ...string) slack.Attachment) func(cmd slack.SlashCommand, c *socketmode.Client) {
	return func(cmd slack.SlashCommand, c *socketmode.Client) {
		args, _ := tokenize(cmd.Text)
		a := f(context.Background(), args...)
		if _, _, err := c.PostMessage(cmd.ChannelID, slack.MsgOptionAttachments(a)); err != nil {
			b.logger.Error("failed to post response", "err", err)
		}
	}
}

// tokenize splits a string into words, preserving quoted phrases as single elements
func tokenize(input string) ([]string, error) {
	var result []string
	var currentWord strings.Builder
	inQuotes := false

	for _, r := range input {
		switch {
		case unicode.IsSpace(r) && !inQuotes:
			// If we're outside quotes and hit a space, finish the current word
			if currentWord.Len() > 0 {
				result = append(result, currentWord.String())
				currentWord.Reset()
			}
		case r == '"', r == '\'':
			// Toggle the inQuotes state
			inQuotes = !inQuotes
			if !inQuotes {
				// If closing a quote, save the quoted word
				result = append(result, currentWord.String())
				currentWord.Reset()
			}
		default:
			// Append the character to the current word
			currentWord.WriteRune(r)
		}
	}

	// Check for unbalanced quotes
	if inQuotes {
		return nil, fmt.Errorf("mismatched quotes in input")
	}

	// Add the last word if there is one
	if currentWord.Len() > 0 {
		result = append(result, currentWord.String())
	}

	return result, nil
}
