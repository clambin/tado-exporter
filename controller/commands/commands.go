package commands

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/slackbot"
	"github.com/clambin/tado-exporter/controller/zonemanager"
	"github.com/clambin/tado-exporter/poller"
	"github.com/slack-go/slack"
	"golang.org/x/exp/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	API    TadoSetter
	poller poller.Poller
	update *poller.Update
	mgrs   zonemanager.Managers
	lock   sync.RWMutex
}

//go:generate mockery --name TadoSetter
type TadoSetter interface {
	DeleteZoneOverlay(context.Context, int) error
	SetZoneTemporaryOverlay(context.Context, int, float64, time.Duration) error
}

func New(api TadoSetter, tadoBot slackbot.SlackBot, p poller.Poller, mgrs zonemanager.Managers) *Manager {
	m := &Manager{
		API:    api,
		poller: p,
		mgrs:   mgrs,
	}

	tadoBot.Register("rules", m.ReportRules)
	tadoBot.Register("rooms", m.ReportRooms)
	tadoBot.Register("set", m.SetRoom)
	tadoBot.Register("refresh", m.DoRefresh)
	tadoBot.Register("users", m.ReportUsers)

	return m
}

// Run the controller
func (m *Manager) Run(ctx context.Context) {
	slog.Info("commands manager started")
	ch := m.poller.Register()
	defer m.poller.Unregister(ch)
	for {
		select {
		case <-ctx.Done():
			slog.Info("commands manager stopped")
			return
		case update := <-ch:
			m.lock.Lock()
			m.update = update
			m.lock.Unlock()
		}
	}
}

func (m *Manager) ReportRules(_ context.Context, _ ...string) []slack.Attachment {
	text, ok := m.mgrs.ReportTasks()

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

func (m *Manager) ReportRooms(_ context.Context, _ ...string) []slack.Attachment {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.update == nil {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "no updates yet. please check back later",
		}}
	}

	text := make([]string, 0)

	for zoneID, zone := range m.update.Zones {
		zoneInfo, found := m.update.ZoneInfo[zoneID]
		if !found {
			continue
		}

		var stateStr string
		switch poller.GetZoneState(zoneInfo) {
		case poller.ZoneStateOff:
			stateStr = "off"
		case poller.ZoneStateAuto:
			stateStr = fmt.Sprintf("target: %.1f", zoneInfo.Setting.Temperature.Celsius)
		case poller.ZoneStateTemporaryManual:
			stateStr = fmt.Sprintf("target: %.1f, MANUAL for %s", zoneInfo.Overlay.Setting.Temperature.Celsius,
				(time.Duration(zoneInfo.Overlay.Termination.RemainingTimeInSeconds) * time.Second).String())
		case poller.ZoneStateManual:
			stateStr = fmt.Sprintf("target: %.1f, MANUAL", zoneInfo.Overlay.Setting.Temperature.Celsius)
		}

		text = append(text, fmt.Sprintf("%s: %.1fºC (%s)", zone.Name, zoneInfo.SensorDataPoints.InsideTemperature.Celsius, stateStr))
	}

	slackColor := "bad"
	slackTitle := ""
	slackText := "no rooms found"

	if len(text) > 0 {
		slackColor = "good"
		slackTitle = "rooms:"
		sort.Strings(text)
		slackText = strings.Join(text, "\n")
	}

	return []slack.Attachment{{
		Color: slackColor,
		Title: slackTitle,
		Text:  slackText,
	}}
}

func (m *Manager) SetRoom(ctx context.Context, args ...string) []slack.Attachment {
	m.lock.RLock()
	defer m.lock.RUnlock()

	zoneID, zoneName, auto, temperature, duration, err := m.parseSetCommand(args...)

	if err != nil {
		err = fmt.Errorf("invalid command: %w", err)
	}

	if err == nil {
		if auto {
			err = m.API.DeleteZoneOverlay(ctx, zoneID)
			if err != nil {
				err = fmt.Errorf("unable to move room to auto mode: %w", err)
			}
		} else {
			err = m.API.SetZoneTemporaryOverlay(ctx, zoneID, temperature, duration)
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

	m.poller.Refresh()

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

func (m *Manager) parseSetCommand(args ...string) (zoneID int, zoneName string, auto bool, temperature float64, duration time.Duration, err error) {
	if len(args) < 2 {
		err = fmt.Errorf("missing parameters\nUsage: set <room> [auto|<temperature> [<duration>]")
		return
	}

	zoneName = args[0]

	var found bool
	for id, zone := range m.update.Zones {
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

func (m *Manager) DoRefresh(_ context.Context, _ ...string) []slack.Attachment {
	m.poller.Refresh()
	return []slack.Attachment{{
		Text: "refreshing Tado data",
	}}
}

func (m *Manager) ReportUsers(_ context.Context, _ ...string) []slack.Attachment {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.update == nil {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "no update yet. please check back later",
		}}
	}

	text := make([]string, 0)

	for _, device := range m.update.UserInfo {
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
