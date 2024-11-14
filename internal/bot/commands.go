package bot

import (
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"log/slog"
	"slices"
	"strings"
	"time"
)

var (
	ErrNoUpdates = errors.New("no updates yet. please check back later")
)

type commandRunner struct {
	TadoClient
	updateStore
	poller     poller.Poller
	controller Controller
	logger     *slog.Logger
}

func (r *commandRunner) dispatch(command slack.SlashCommand, client SlackSender) error {
	r.logger.Debug("running command", "cmd", command.Command, "text", command.Text)
	var err error
	switch command.Text {
	case "rooms":
		err = r.listRooms(command, client)
	case "users":
		err = r.listUsers(command, client)
	case "rules":
		err = r.listRules(command, client)
	case "refresh":
		err = r.refresh(command, client)
	case "help":
		err = r.help(command, client)
	default:
		err = errors.New("unknown command: " + command.Text)
	}
	return err
}

func (r *commandRunner) listRooms(command slack.SlashCommand, client SlackSender) error {
	u, ok := r.getUpdate()
	if !ok {
		return ErrNoUpdates
	}

	text := make([]string, 0, len(u.Zones))

	for _, zone := range u.Zones {
		text = append(text, fmt.Sprintf("%s: %.1fºC (%s)",
			*zone.Name,
			*zone.SensorDataPoints.InsideTemperature.Celsius,
			zoneState(zone),
		))
	}

	if len(text) == 0 {
		return errors.New("no rooms found")
	}

	slices.Sort(text)
	attachment := slack.Attachment{
		Color: "good",
		Title: "rooms",
		Text:  strings.Join(text, "\n"),
	}
	_, err := client.PostEphemeral(command.ChannelID, command.UserID, slack.MsgOptionAttachments(attachment))
	return err
}

func zoneState(zone poller.Zone) string {
	targetTemperature := zone.GetTargetTemperature()
	if targetTemperature == 0.0 {
		return "off"
	}

	if zone.Overlay == nil {
		return fmt.Sprintf("target: %.1f", targetTemperature)
	}
	switch *zone.Overlay.Termination.Type {
	case tado.ZoneOverlayTerminationTypeMANUAL:
		return fmt.Sprintf("target: %.1f, MANUAL", targetTemperature)
	default:
		return fmt.Sprintf("target: %.1f, MANUAL for %s", targetTemperature, (time.Duration(*zone.Overlay.Termination.RemainingTimeInSeconds) * time.Second).String())
	}
}

func (r *commandRunner) listUsers(command slack.SlashCommand, client SlackSender) error {
	u, ok := r.getUpdate()
	if !ok {
		return ErrNoUpdates
	}

	if len(u.MobileDevices) == 0 {
		return errors.New("no users found")
	}

	text := make([]string, 0)

	for device := range u.MobileDevices.GeoTrackedDevices() {
		location := map[bool]string{true: "home", false: "away"}[*device.Location.AtHome]
		text = append(text, *device.Name+": "+location)
	}
	slices.Sort(text)

	attachment := slack.Attachment{
		Color: "good",
		Title: "users",
		Text:  strings.Join(text, "\n"),
	}

	_, err := client.PostEphemeral(command.ChannelID, command.UserID, slack.MsgOptionAttachments(attachment))
	return err
}

func (r *commandRunner) listRules(command slack.SlashCommand, client SlackSender) error {
	r.lock.RLock()
	defer r.lock.RUnlock()
	// a bit icky, but okay ...
	if r.controller == nil || r.controller.(*mocks.Controller) == nil {
		return errors.New("controller isn't running")
	}

	text := "no rules have been triggered"
	if rules := r.controller.ReportTasks(); len(rules) != 0 {
		text = strings.Join(rules, "\n")
	}
	attachment := slack.Attachment{
		Color: "good",
		Title: "rules",
		Text:  text,
	}
	_, err := client.PostEphemeral(command.ChannelID, command.UserID, slack.MsgOptionAttachments(attachment))
	return err
}

func (r *commandRunner) refresh(command slack.SlashCommand, client SlackSender) error {
	r.poller.Refresh()
	_, err := client.PostEphemeral(command.ChannelID, command.UserID, slack.MsgOptionText("refreshing Tadoº data", false))
	return err
}

func (r *commandRunner) help(command slack.SlashCommand, client SlackSender) error {
	help := "supported commands: users, rooms, rules, help"
	_, err := client.PostEphemeral(command.ChannelID, command.UserID, slack.MsgOptionText(help, false))
	return err
}
