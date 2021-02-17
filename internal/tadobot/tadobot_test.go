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

	assert.Equal(t, "tado v"+version.BuildVersion, responses[0])
}

func TestTadoBot_DoRooms(t *testing.T) {
	bot := TadoBot{
		API: &mockapi.MockAPI{},
	}

	responses := bot.doRooms()
	sort.Strings(responses)

	assert.Equal(t, "bar: 19.9ºC (target: 25.0ºC MANUAL)", responses[0])
	assert.Equal(t, "foo: 19.9ºC (target: 20.0ºC MANUAL)", responses[1])

}

func TestTadoBot_DoUsers(t *testing.T) {
	bot := TadoBot{
		API: &mockapi.MockAPI{},
	}

	responses := bot.doUsers()
	sort.Strings(responses)

	assert.Equal(t, "bar: away", responses[0])
	assert.Equal(t, "foo: home", responses[1])
}

func TestTadoBot_ProcessMessage(t *testing.T) {
	bot, _ := Create("", "", "", "")
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
	assert.Equal(t, "tado v"+version.BuildVersion, attachment.Text)
}
