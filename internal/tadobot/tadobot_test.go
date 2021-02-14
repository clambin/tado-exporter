package tadobot_test

import (
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTadoBot_DoVersion(t *testing.T) {
	bot := tadobot.TadoBot{}

	response := bot.DoVersion()

	assert.Equal(t, "tado v"+version.BuildVersion, response)
}

func TestTadoBot_DoRooms(t *testing.T) {
	bot := tadobot.TadoBot{
		API: &mockapi.MockAPI{},
	}

	response := bot.DoRooms()

	assert.Equal(t, "foo: 19.9ºC (target: 20.0ºC MANUAL)\nbar: 19.9ºC (target: 25.0ºC MANUAL)", response)
}

func TestTadoBot_DoUsers(t *testing.T) {
	bot := tadobot.TadoBot{
		API: &mockapi.MockAPI{},
	}

	response := bot.DoUsers()

	assert.Equal(t, "foo: home\nbar: away", response)
}
