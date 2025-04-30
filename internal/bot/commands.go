package bot

import (
	"cmp"
	"errors"
	"fmt"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/slacktools"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"log/slog"
	"slices"
	"time"
)

var (
	ErrNoUpdates = errors.New("no updates yet. please check back later")
)

type commandRunner struct {
	tadoClient TadoClient
	poller     poller.Poller
	controller Controller
	logger     *slog.Logger
	updateStore
}

func (r *commandRunner) dispatch(command slack.SlashCommand, client SlackSender) error {
	r.logger.Debug("running command", "cmd", command.Command, "text", command.Text)
	var response slacktools.Formatter
	var err error
	switch command.Text {
	case "rooms":
		response, err = r.listRooms()
	case "users":
		response, err = r.listUsers()
	case "rules":
		response, err = r.listRules()
	case "refresh":
		response, err = r.refresh()
	case "help":
		response, err = r.help()
	default:
		err = errors.New("unknown command: " + command.Text)
	}
	if err == nil && response != nil && !response.IsZero() {
		_, err = client.PostEphemeral(command.ChannelID, command.UserID, response.Format())
	}
	return err
}

func (r *commandRunner) listRooms() (slacktools.Attachment, error) {
	u, ok := r.getUpdate()
	if !ok {
		return slacktools.Attachment{}, ErrNoUpdates
	}

	lines := make([]string, 0, cmp.Or(len(u.Zones), 1))
	for _, zone := range u.Zones {
		lines = append(lines, fmt.Sprintf("*%s*: %.1fºC (%s)",
			*zone.Zone.Name,
			*zone.ZoneState.SensorDataPoints.InsideTemperature.Celsius,
			zoneState(zone),
		))
	}
	slices.Sort(lines)

	if len(lines) == 0 {
		lines = append(lines, "no rooms have been found")
	}

	return slacktools.Attachment{Header: "Rooms:", Body: lines}, nil
}

func zoneState(zone poller.Zone) string {
	targetTemperature := zone.GetTargetTemperature()
	if targetTemperature == 0.0 {
		return "off"
	}

	if zone.ZoneState.Overlay == nil {
		return fmt.Sprintf("target: %.1f", targetTemperature)
	}
	switch *zone.ZoneState.Overlay.Termination.Type {
	case tado.ZoneOverlayTerminationTypeMANUAL:
		return fmt.Sprintf("target: %.1f, MANUAL", targetTemperature)
	default:
		return fmt.Sprintf("target: %.1f, MANUAL for %s", targetTemperature, (time.Duration(*zone.ZoneState.Overlay.Termination.RemainingTimeInSeconds) * time.Second).String())
	}
}

func (r *commandRunner) listUsers() (slacktools.Attachment, error) {
	u, ok := r.getUpdate()
	if !ok {
		return slacktools.Attachment{}, ErrNoUpdates
	}

	lines := make([]string, 0, len(u.MobileDevices))

	for device := range u.MobileDevices.GeoTrackedDevices() {
		location := map[bool]string{true: "home", false: "away"}[*device.Location.AtHome]
		lines = append(lines, "*"+*device.Name+"*: "+location)
	}
	slices.Sort(lines)

	if len(lines) == 0 {
		lines = append(lines, "no users have been found")
	}

	return slacktools.Attachment{Header: "Users:", Body: lines}, nil
}

func (r *commandRunner) listRules() (slacktools.Attachment, error) {
	if r.controller == nil {
		return slacktools.Attachment{}, errors.New("controller isn't running")
	}
	rules := r.controller.ReportTasks()
	if len(rules) == 0 {
		rules = []string{"no rules have been triggered"}
	} else {
		slices.Sort(rules)
	}
	return slacktools.Attachment{Header: "Rules:", Body: rules}, nil
}

func (r *commandRunner) refresh() (slacktools.Attachment, error) {
	r.poller.Refresh()
	return slacktools.Attachment{}, nil
}

func (r *commandRunner) help() (slacktools.Attachment, error) {
	return slacktools.Attachment{Header: "Supported commands:", Body: []string{"users, rooms, rules, help"}}, nil
}
