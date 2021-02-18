package tadobot

import (
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestTadoBot_DoVersion(t *testing.T) {
	bot := TadoBot{}

	responses := bot.doVersion()
	sort.Strings(responses)

	assert.Equal(t, "tado "+version.BuildVersion, responses[0])
}

func TestTadoBot_ProcessMessage(t *testing.T) {
	bot, _ := Create("", "", "", "", nil)
	bot.API = &mockapi.MockAPI{}
	bot.userID = "12345678"

	var attachment *slack.Attachment

	attachment = bot.processMessage("Hello world")
	assert.Nil(t, attachment)
	attachment = bot.processMessage("<@12345678> Hello")
	assert.NotNil(t, attachment)
	assert.Equal(t, "Unknown command \"Hello\"", attachment.Title)
	attachment = bot.processMessage("<@12345678> version")
	assert.NotNil(t, attachment)
	assert.Equal(t, "tado "+version.BuildVersion, attachment.Text)
}
