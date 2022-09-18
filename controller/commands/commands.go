package commands

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/clambin/tado-exporter/slackbot"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	tado.API
	poller  poller.Poller
	update  *poller.Update
	updates chan *poller.Update
	lock    sync.RWMutex
}

func New(api tado.API, tadoBot slackbot.SlackBot, p poller.Poller) *Manager {
	m := &Manager{
		API:     api,
		poller:  p,
		updates: make(chan *poller.Update),
	}

	//tadoBot.RegisterCallback("rules", m.ReportRules)
	tadoBot.RegisterCallback("rooms", m.ReportRooms)
	tadoBot.RegisterCallback("set", m.SetRoom)
	tadoBot.RegisterCallback("refresh", m.DoRefresh)
	tadoBot.RegisterCallback("users", m.ReportUsers)

	return m
}

// Run the controller
func (m *Manager) Run(ctx context.Context) {
	log.Info("commands manager started")

	m.poller.Register(m.updates)
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case update := <-m.updates:
			m.lock.Lock()
			m.update = update
			m.lock.Unlock()
		}
	}
	m.poller.Unregister(m.updates)
	log.Info("commands manager stopped")
}

/*
func (m *Manager) ReportRules(_ context.Context, _ ...string) (attachments []slack.Attachment) {
	text := make([]string, 0)
	for zoneID, scheduled := range m.Setter.GetScheduled() {
		var action string
		switch scheduled.State {
		case tado.ZoneStateOff:
			action = "switching off heating"
		case tado.ZoneStateAuto:
			action = "moving to auto mode"
			//case tado.ZoneStateManual:
			//	action = "setting to manual temperature control"
		}

		name, _ := m.cache.GetZoneName(zoneID)

		text = append(text,
			name+": "+action+" in "+time.Until(scheduled.When).Round(1*time.Second).String())
	}

	var slackText, slackTitle string
	if len(text) > 0 {
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
*/

func (m *Manager) ReportRooms(_ context.Context, _ ...string) (attachments []slack.Attachment) {
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
		switch zoneInfo.GetState() {
		case tado.ZoneStateOff:
			stateStr = "off"
		case tado.ZoneStateAuto:
			stateStr = fmt.Sprintf("target: %.1f", zoneInfo.Setting.Temperature.Celsius)
		case tado.ZoneStateTemporaryManual:
			stateStr = fmt.Sprintf("target: %.1f, MANUAL for %s", zoneInfo.Overlay.Setting.Temperature.Celsius,
				(time.Duration(zoneInfo.Overlay.Termination.RemainingTime) * time.Second).String())
		case tado.ZoneStateManual:
			stateStr = fmt.Sprintf("target: %.1f, MANUAL", zoneInfo.Overlay.Setting.Temperature.Celsius)
		}

		text = append(text, fmt.Sprintf("%s: %.1fºC (%s)", zone.Name, zoneInfo.SensorDataPoints.Temperature.Celsius, stateStr))
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

func (m *Manager) SetRoom(ctx context.Context, args ...string) (attachments []slack.Attachment) {
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
			err = m.API.SetZoneOverlayWithDuration(ctx, zoneID, temperature, duration)
			if err != nil {
				err = fmt.Errorf("unable to set temperature for %s: %w", zoneName, err)
			}
		}
	}

	if err != nil {
		attachments = []slack.Attachment{{
			Color: "bad",
			Title: "",
			Text:  err.Error(),
		}}
	} else {
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
		attachments = []slack.Attachment{{
			Color: "good",
			Title: "",
			Text:  text,
		}}
	}

	return
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

func (m *Manager) DoRefresh(_ context.Context, _ ...string) (attachments []slack.Attachment) {
	m.poller.Refresh()
	return []slack.Attachment{{
		Text: "refreshing Tado data",
	}}
}

func (m *Manager) ReportUsers(_ context.Context, _ ...string) (attachments []slack.Attachment) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.update == nil {
		return []slack.Attachment{{
			Color: "bad",
			Text:  "no update yet. please check back later",
		}}
	}

	text := make([]string, 0)

	for _, device := range m.update.UserInfo { // m.cache.GetUsers() {
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