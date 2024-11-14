package bot

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/tadotools"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"log/slog"
	"strconv"
	"time"
)

const (
	setRoomCallbackID = "tado_set_room"
	setHomeCallbackID = "tado_set_home"
)

type shortcuts map[string]shortcutHandler

type shortcutHandler interface {
	HandleShortcut(slack.InteractionCallback, SlackSender) error
	HandleAction(slack.InteractionCallback, SlackSender) error
	HandleSubmission(slack.InteractionCallback, SlackSender) error
	setUpdate(poller.Update)
}

func (s shortcuts) dispatch(data slack.InteractionCallback, client SlackSender) error {
	var callbackID string
	switch data.Type {
	case slack.InteractionTypeShortcut:
		callbackID = data.CallbackID
	case slack.InteractionTypeBlockActions, slack.InteractionTypeViewSubmission:
		callbackID = data.View.CallbackID
	}

	handler, ok := s[callbackID]
	if !ok {
		return fmt.Errorf("unknown callbackID: %q", data.CallbackID)
	}
	switch data.Type {
	case slack.InteractionTypeShortcut:
		return handler.HandleShortcut(data, client)
	case slack.InteractionTypeBlockActions:
		return handler.HandleAction(data, client)
	case slack.InteractionTypeViewSubmission:
		return handler.HandleSubmission(data, client)
	default:
		return nil
	}
}

func (s shortcuts) setUpdate(u poller.Update) {
	for _, h := range s {
		h.setUpdate(u)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ shortcutHandler = &setRoomShortcut{}

type setRoomShortcut struct {
	TadoClient
	updateStore
	logger *slog.Logger
}

func (s *setRoomShortcut) HandleShortcut(event slack.InteractionCallback, client SlackSender) error {
	u, _ := s.getUpdate()
	resp, err := client.OpenView(event.TriggerID, s.makeView("auto", u))
	if err != nil {
		return fmt.Errorf("failed to open view for %q: %w", event.CallbackID, err)
	}
	s.logger.Debug("opened view", "callbackID", resp.CallbackID, "viewID", resp.ID, "externalID", resp.ExternalID)
	return nil
}

func (s *setRoomShortcut) HandleAction(data slack.InteractionCallback, client SlackSender) error {
	for _, a := range data.ActionCallback.BlockActions {
		s.logger.Debug("action received", "viewID", data.View.ID, "actionID", a.ActionID, "blockID", a.BlockID, "value", a.SelectedOption.Value)
		switch a.ActionID {
		case "mode":
			u, _ := s.getUpdate()
			_, err := client.UpdateView(s.makeView(a.SelectedOption.Value, u), "", "", data.View.ID)
			if err != nil {
				return fmt.Errorf("failed to update view: %w", err)
			}
		default:
			return fmt.Errorf("unknown actionID: %q/%q", a.BlockID, a.ActionID)
		}
	}
	return nil
}

func (s *setRoomShortcut) HandleSubmission(data slack.InteractionCallback, client SlackSender) error {
	var postErr error
	channel, action, err := s.setRoom(data)
	if err == nil {
		_, _, postErr = client.PostMessage(channel, slack.MsgOptionText("<@"+data.User.ID+"> "+action, false))
	} else {
		_, postErr = client.PostEphemeral(channel, data.User.ID, slack.MsgOptionText("failed to set room: "+err.Error(), false))
	}
	if postErr != nil {
		s.logger.Warn("failed to post message", "err", postErr)
	}
	return err
}

func (s *setRoomShortcut) setRoom(data slack.InteractionCallback) (string, string, error) {
	zoneName := data.View.State.Values["zone"]["zone"].SelectedOption.Value
	mode := data.View.State.Values["mode"]["mode"].SelectedOption.Value
	temperature := data.View.State.Values["temperature"]["temperature"].Value
	due := data.View.State.Values["expiration"]["expiration"].SelectedTime
	channel := data.View.State.Values["channel"]["channel"].SelectedConversation

	u, _ := s.getUpdate()
	homeId := *u.HomeBase.Id
	zone, _ := u.GetZone(zoneName)

	var err error
	var action string
	ctx := context.Background()
	switch mode {
	case "manual":
		var duration time.Duration
		temp, _ := strconv.ParseFloat(temperature, 32)
		action = "set *" + zoneName + "* to " + temperature + "ºC"
		if due != "" {
			if duration, err = timeStampToDuration(due); err != nil {
				return "", "", fmt.Errorf("failed to parse due time %q: %w", due, err)
			}
			action += " for " + duration.Round(time.Minute).String()
		}
		err = tadotools.SetOverlay(ctx, s.TadoClient, homeId, *zone.Id, float32(temp), duration)
	case "auto":
		action = "set *" + zoneName + "* to auto mode"
		_, err = s.TadoClient.DeleteZoneOverlayWithResponse(ctx, homeId, *zone.Id)
	default:
	}

	return channel, action, err
}

func (s *setRoomShortcut) makeView(mode string, u poller.Update) slack.ModalViewRequest {
	zones := make([]string, len(u.Zones))
	for i, z := range u.Zones {
		zones[i] = *z.Name
	}

	blocks := slack.Blocks{BlockSet: make([]slack.Block, 2, 4)}

	// zone
	zoneLabel := slack.NewTextBlockObject(slack.PlainTextType, "Zone:", false, false)
	zoneOptions := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, nil, "zone", createOptionBlockObjects(zones...)...)
	zoneBlock := slack.NewInputBlock("zone", zoneLabel, nil, zoneOptions)
	blocks.BlockSet[0] = zoneBlock

	// mode
	modeLabel := slack.NewTextBlockObject(slack.PlainTextType, "Mode:", false, false)
	modeOptions := slack.NewRadioButtonsBlockElement("mode", createOptionBlockObjects("auto", "manual")...)
	modeBlock := slack.NewInputBlock("mode", modeLabel, nil, modeOptions).WithDispatchAction(true)
	blocks.BlockSet[1] = modeBlock

	if mode == "manual" {
		// temperature
		temperatureLabel := slack.NewTextBlockObject(slack.PlainTextType, "Temperature:", false, false)
		temperatureOptions := slack.NewNumberInputBlockElement(nil, "temperature", true)
		temperatureBlock := slack.NewInputBlock("temperature", temperatureLabel, nil, temperatureOptions)
		blocks.BlockSet = append(blocks.BlockSet, temperatureBlock)

		// duration
		// TODO: duration -> expiration
		durationLabel := slack.NewTextBlockObject(slack.PlainTextType, "Expiration:", false, false)
		durationOptions := slack.NewTimePickerBlockElement("expiration")
		durationBlock := slack.NewInputBlock("expiration", durationLabel, nil, durationOptions).WithOptional(true)

		blocks.BlockSet = append(blocks.BlockSet, durationBlock)
	}

	// channel to respond
	channelLabel := slack.NewTextBlockObject(slack.PlainTextType, "Channel:", false, false)
	//channelOptions := slack.NewConversationsSelect("channel", "Channel:")
	channelSelect := slack.NewOptionsSelectBlockElement(slack.OptTypeConversations, nil, "channel")
	channelSelect.DefaultToCurrentConversation = true
	channelBlock := slack.NewInputBlock("channel", channelLabel, nil, channelSelect)
	blocks.BlockSet = append(blocks.BlockSet, channelBlock)

	return slack.ModalViewRequest{
		Type:          slack.VTModal,
		Title:         slack.NewTextBlockObject(slack.PlainTextType, "Set Room", false, false),
		Blocks:        blocks,
		Close:         slack.NewTextBlockObject(slack.PlainTextType, "Close", false, false),
		Submit:        slack.NewTextBlockObject(slack.PlainTextType, "Submit", false, false),
		CallbackID:    setRoomCallbackID,
		ClearOnClose:  false,
		NotifyOnClose: false,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ shortcutHandler = &setHomeShortcut{}

type setHomeShortcut struct {
	TadoClient
	updateStore
	logger *slog.Logger
}

func (s *setHomeShortcut) HandleShortcut(event slack.InteractionCallback, client SlackSender) error {
	resp, err := client.OpenView(event.TriggerID, s.makeView())
	if err != nil {
		return fmt.Errorf("failed to open view for %q: %w", event.CallbackID, err)
	}
	s.logger.Debug("opened view", "callbackID", resp.CallbackID, "viewID", resp.ID, "externalID", resp.ExternalID)
	return nil
}

func (s *setHomeShortcut) HandleAction(_ slack.InteractionCallback, _ SlackSender) error {
	return nil
}

func (s *setHomeShortcut) HandleSubmission(data slack.InteractionCallback, client SlackSender) error {
	var postErr error
	channel, action, err := s.setHome(data)
	if err == nil {
		_, _, postErr = client.PostMessage(channel, slack.MsgOptionText("<@"+data.User.ID+"> "+action, false))
	} else {
		_, postErr = client.PostEphemeral(channel, data.User.ID, slack.MsgOptionText("failed to set home: "+err.Error(), false))
	}
	if postErr != nil {
		s.logger.Warn("failed to post message", "err", postErr)
	}
	return err
}

func (s *setHomeShortcut) setHome(data slack.InteractionCallback) (string, string, error) {
	mode := data.View.State.Values["mode"]["mode"].SelectedOption.Value
	channel := data.View.State.Values["channel"]["channel"].SelectedConversation
	action := "set home to " + mode + " mode"

	u, _ := s.getUpdate()
	homeId := *u.HomeBase.Id

	var err error
	ctx := context.Background()
	switch mode {
	case "auto":
		_, err = s.TadoClient.DeletePresenceLockWithResponse(ctx, homeId)
	case "home":
		_, err = s.TadoClient.SetPresenceLockWithResponse(ctx, homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: oapi.VarP(tado.HOME)})
	case "away":
		_, err = s.TadoClient.SetPresenceLockWithResponse(ctx, homeId, tado.SetPresenceLockJSONRequestBody{HomePresence: oapi.VarP(tado.AWAY)})
	}
	return channel, action, err
}

func (s *setHomeShortcut) makeView() slack.ModalViewRequest {
	blocks := slack.Blocks{BlockSet: make([]slack.Block, 2)}

	// mode
	modeLabel := slack.NewTextBlockObject(slack.PlainTextType, "Mode:", false, false)
	modeOptions := slack.NewRadioButtonsBlockElement("mode", createOptionBlockObjects("auto", "home", "away")...)
	modeBlock := slack.NewInputBlock("mode", modeLabel, nil, modeOptions)
	blocks.BlockSet[0] = modeBlock

	// channel to post response
	channelLabel := slack.NewTextBlockObject(slack.PlainTextType, "Channel:", false, false)
	channelSelect := slack.NewOptionsSelectBlockElement(slack.OptTypeConversations, nil, "channel")
	channelSelect.DefaultToCurrentConversation = true
	channelBlock := slack.NewInputBlock("channel", channelLabel, nil, channelSelect)
	blocks.BlockSet[1] = channelBlock

	return slack.ModalViewRequest{
		Type:          slack.VTModal,
		Title:         slack.NewTextBlockObject(slack.PlainTextType, "Set Room", false, false),
		Blocks:        blocks,
		Close:         slack.NewTextBlockObject(slack.PlainTextType, "Close", false, false),
		Submit:        slack.NewTextBlockObject(slack.PlainTextType, "Submit", false, false),
		CallbackID:    setHomeCallbackID,
		ClearOnClose:  false,
		NotifyOnClose: false,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func createOptionBlockObjects(options ...string) []*slack.OptionBlockObject {
	optionBlockObjects := make([]*slack.OptionBlockObject, len(options))
	for i, o := range options {
		optionText := slack.NewTextBlockObject(slack.PlainTextType, o, false, false)
		optionBlockObjects[i] = slack.NewOptionBlockObject(o, optionText, nil)
	}
	return optionBlockObjects
}

var nowFunc = time.Now

func timeStampToDuration(targetTime string) (time.Duration, error) {
	// allow tests to override current time
	now := nowFunc()
	loc := now.Location()

	// Parse the target time
	parsedTime, err := time.ParseInLocation("15:04", targetTime, loc)
	if err != nil {
		return 0, err
	}

	// Set the target time to today’s date with the parsed hour and minute
	target := time.Date(now.Year(), now.Month(), now.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)

	// If the target time is before the current time, move it to the next day
	if target.Before(now) {
		target = target.Add(24 * time.Hour)
	}

	// Calculate the duration until the target time
	duration := target.Sub(now)
	return duration, nil
}
