package controller

/*
import (
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/slack-go/slack"
	"sort"
	"strconv"
	"strings"
)

func (controller *Controller) doRooms(_ ...string) []slack.Attachment {
	output := make([]string, 0)
	zoneNames := controller.proxy.GetAllZones()
	for id, zoneState := range controller.proxy.GetAllZoneStates() {
		zoneName := zoneNames[id]
		output = append(output, zoneName+": "+zoneState.String())
	}
	sort.Strings(output)
	return []slack.Attachment{
		{
			Color: "good",
			Title: "Rooms:",
			Text:  strings.Join(output, "\n"),
		},
	}
}


func (controller *Controller) doUsers(_ ...string) []slack.Attachment {
	output := make([]string, 0)
	userNames := controller.proxy.GetAllUsers()
	for id, userState := range controller.proxy.GetAllUserStates() {
		userName := userNames[id]
		output = append(output, userName+": "+userState.String())
	}
	sort.Strings(output)
	return []slack.Attachment{
		{
			Color: "good",
			Title: "Users:",
			Text:  strings.Join(output, "\n"),
		},
	}
}

*/
