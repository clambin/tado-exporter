package commands

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/internal/controller/slackbot"
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

type Executor struct {
	Tado        TadoSetter
	poller      poller.Poller
	update      *poller.Update
	controllers Controllers
	logger      *slog.Logger
	lock        sync.RWMutex
}

type Controllers interface {
	ReportTasks() ([]string, bool)
}

type TadoSetter interface {
	DeleteZoneOverlay(context.Context, int) error
	SetZoneTemporaryOverlay(context.Context, int, float64, time.Duration) error
}

func New(tado TadoSetter, tadoBot slackbot.SlackBot, p poller.Poller, controllers Controllers, logger *slog.Logger) *Executor {
	executor := Executor{
		Tado:        tado,
		poller:      p,
		controllers: controllers,
		logger:      logger,
	}

	tadoBot.Register("rules", executor.ReportRules)
	tadoBot.Register("rooms", executor.ReportRooms)
	tadoBot.Register("set", executor.SetRoom)
	tadoBot.Register("refresh", executor.DoRefresh)
	tadoBot.Register("users", executor.ReportUsers)

	return &executor
}

// Run the controller
func (e *Executor) Run(ctx context.Context) error {
	e.logger.Debug("started")
	defer e.logger.Debug("stopped")

	ch := e.poller.Subscribe()
	defer e.poller.Unsubscribe(ch)
	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-ch:
			e.lock.Lock()
			e.update = update
			e.lock.Unlock()
		}
	}
}

func (e *Executor) ReportRules(_ context.Context, _ ...string) []slack.Attachment {
	text, ok := e.controllers.ReportTasks()

	var slackText, slackTitle string
	if ok {
		slackTitle = "rules:"
		slackText = strings.Join(text, "\n")
	} else {
		slackTitle = ""
		slackText = "no rules have been triggered"
	}

	return []slack.Attachment{{
		Color: "good",
		Title: slackTitle,
		Text:  slackText,
	}}
}

func (e *Executor) ReportRooms(_ context.Context, _ ...string) []slack.Attachment {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if e.update == nil {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "no updates yet. please check back later",
		}}
	}

	text := make([]string, 0)

	for zoneID, zone := range e.update.Zones {
		zoneInfo, found := e.update.ZoneInfo[zoneID]
		if !found {
			continue
		}

		var stateStr string
		zoneState := rules.GetZoneState(zoneInfo)
		if !zoneState.Heating() {
			stateStr = "off"
		} else {
			switch zoneState.Overlay {
			case tado.NoOverlay:
				stateStr = fmt.Sprintf("target: %.1f", zoneState.TargetTemperature.Celsius)
			case tado.PermanentOverlay:
				stateStr = fmt.Sprintf("target: %.1f, MANUAL", zoneInfo.Setting.Temperature.Celsius)
			case tado.TimerOverlay, tado.NextBlockOverlay:
				stateStr = fmt.Sprintf("target: %.1f, MANUAL for %s", zoneInfo.Setting.Temperature.Celsius,
					(time.Duration(zoneInfo.Overlay.Termination.RemainingTimeInSeconds) * time.Second).String())
			}
		}

		text = append(text, fmt.Sprintf("%s: %.1fºC (%s)", zone.Name, zoneInfo.SensorDataPoints.InsideTemperature.Celsius, stateStr))
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

func (e *Executor) SetRoom(ctx context.Context, args ...string) []slack.Attachment {
	e.lock.RLock()
	defer e.lock.RUnlock()

	zoneID, zoneName, auto, temperature, duration, err := e.parseSetCommand(args...)

	if err != nil {
		err = fmt.Errorf("invalid command: %w", err)
	}

	if err == nil {
		if auto {
			err = e.Tado.DeleteZoneOverlay(ctx, zoneID)
			if err != nil {
				err = fmt.Errorf("unable to move room to auto mode: %w", err)
			}
		} else {
			err = e.Tado.SetZoneTemporaryOverlay(ctx, zoneID, temperature, duration)
			if err != nil {
				err = fmt.Errorf("unable to set temperature for %s: %w", zoneName, err)
			}
		}
	}

	if err != nil {
		return []slack.Attachment{{
			Color: "bad",
			Title: "",
			Text:  err.Error(),
		}}
	}

	e.poller.Refresh()

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

func (e *Executor) parseSetCommand(args ...string) (zoneID int, zoneName string, auto bool, temperature float64, duration time.Duration, err error) {
	if len(args) < 2 {
		err = fmt.Errorf("missing parameters\nUsage: set <room> [auto|<temperature> [<duration>]")
		return
	}

	zoneName = args[0]

	var found bool
	for id, zone := range e.update.Zones {
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

func (e *Executor) DoRefresh(_ context.Context, _ ...string) []slack.Attachment {
	e.poller.Refresh()
	return []slack.Attachment{{
		Text: "refreshing Tado data",
	}}
}

func (e *Executor) ReportUsers(_ context.Context, _ ...string) []slack.Attachment {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if e.update == nil {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "no update yet. please check back later",
		}}
	}

	text := make([]string, 0)

	for _, device := range e.update.UserInfo {
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
