package bot

import (
	"github.com/slack-go/slack"
)

const (
	setRoomCallbackID = "tado_set_room"
)

func setRoomView() slack.ModalViewRequest {
	// zone
	zoneLabel := slack.NewTextBlockObject(slack.PlainTextType, "Zone:", false, false)
	zoneOptions := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, nil, "zoneName", createOptionBlockObjects([]string{"Bathroom", "Study"})...)
	zoneBlock := slack.NewInputBlock("zone", zoneLabel, nil, zoneOptions)

	// mode (radio buttons)

	// temperature
	// duration

	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			zoneBlock,
		},
	}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "Set Room", false, false),
		Blocks:          blocks,
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Close", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Submit", false, false),
		PrivateMetadata: "",
		CallbackID:      setRoomCallbackID,
		ClearOnClose:    false,
		NotifyOnClose:   false,
		ExternalID:      "",
	}
}

func createOptionBlockObjects(options []string) []*slack.OptionBlockObject {
	optionBlockObjects := make([]*slack.OptionBlockObject, len(options))
	for i, o := range options {
		optionText := slack.NewTextBlockObject(slack.PlainTextType, o, false, false)
		optionBlockObjects[i] = slack.NewOptionBlockObject(o, optionText, nil)
	}
	return optionBlockObjects
}
