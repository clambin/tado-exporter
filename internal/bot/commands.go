package bot

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/tadotools"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoUpdates = errors.New("no updates yet. please check back later")
)

type commandRunner struct {
	TadoClient
	poller     poller.Poller
	controller Controller
	logger     *slog.Logger
	update     poller.Update
	lock       sync.RWMutex
	updated    bool
}

func (r *commandRunner) setUpdate(update poller.Update) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.update = update
	r.updated = true
}

func (r *commandRunner) getUpdate() (poller.Update, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.update, r.updated
}

func (r *commandRunner) listRooms(command slack.SlashCommand, client SlackSender) error {
	update, ok := r.getUpdate()
	if !ok {
		return ErrNoUpdates
	}

	text := make([]string, 0, len(update.Zones))

	for _, zone := range update.Zones {
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

func (r *commandRunner) listUsers(command slack.SlashCommand, client SlackSender) error {
	update, ok := r.getUpdate()
	if !ok {
		return ErrNoUpdates
	}

	if len(update.MobileDevices) == 0 {
		return errors.New("no users found")
	}

	text := make([]string, 0)

	for device := range update.MobileDevices.GeoTrackedDevices() {
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
	if r.controller == nil {
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

func (r *commandRunner) setRoom(command slack.SlashCommand, client SlackSender) error {
	cmd, err := parseSetRoom(command.Text)
	if err != nil {
		return err
	}

	update, ok := r.getUpdate()
	if !ok {
		return ErrNoUpdates
	}
	zone, err := update.GetZone(cmd.zoneName)
	if err != nil {
		return fmt.Errorf("invalid room name: %q", cmd.zoneName)
	}

	ctx := context.Background()
	if cmd.mode == "auto" {
		_, err = r.TadoClient.DeleteZoneOverlayWithResponse(ctx, *update.HomeBase.Id, *zone.Id)
	} else {
		err = tadotools.SetOverlay(ctx, r.TadoClient, *update.HomeBase.Id, *zone.Id, float32(cmd.temperature), cmd.duration)
	}

	if err != nil {
		return fmt.Errorf("could not set room: %w", err)
	}

	var text string
	if cmd.mode == "auto" {
		text = "set " + cmd.zoneName + " to automatic mode"
	} else {
		text = fmt.Sprintf("set target temperature for %s to %.1fºC", cmd.zoneName, cmd.temperature)
		if cmd.duration > 0 {
			text += " for " + cmd.duration.String()
		}
	}
	text = "<@" + command.UserID + "> " + text

	_, _, err = client.PostMessage(command.ChannelID, slack.MsgOptionText(text, false))
	r.poller.Refresh()
	return err
}

func (r *commandRunner) setHome(command slack.SlashCommand, client SlackSender) error {
	args := tokenizeText(command.Text)
	if len(args) != 1 {
		return errors.New("missing parameter\nUsage: set home [home|away|auto]")
	}

	update, ok := r.getUpdate()
	if !ok {
		return ErrNoUpdates
	}

	var err error
	ctx := context.Background()
	switch args[0] {
	case "home":
		_, err = r.TadoClient.SetPresenceLockWithResponse(ctx, *update.HomeBase.Id, tado.SetPresenceLockJSONRequestBody{HomePresence: oapi.VarP(tado.HOME)})
	case "away":
		_, err = r.TadoClient.SetPresenceLockWithResponse(ctx, *update.HomeBase.Id, tado.SetPresenceLockJSONRequestBody{HomePresence: oapi.VarP(tado.AWAY)})
	case "auto":
		_, err = r.TadoClient.DeletePresenceLockWithResponse(ctx, *update.HomeBase.Id)
	default:
		return errors.New("missing parameter\nUsage: set home [home|away|auto]")
	}

	if err != nil {
		return err
	}

	text := "<@" + command.UserID + "> moves home to " + args[0] + "mode"
	_, _, err = client.PostMessage(command.ChannelID, slack.MsgOptionText(text, false))
	r.poller.Refresh()
	return err
}
