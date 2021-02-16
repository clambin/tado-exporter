package tadobot_test

import (
	"github.com/clambin/tado-exporter/internal/tadobot"
	"github.com/clambin/tado-exporter/internal/version"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestTadoBot_DoVersion(t *testing.T) {
	bot := tadobot.TadoBot{}

	responses := bot.DoVersion()
	sort.Strings(responses)

	assert.Equal(t, "tado v"+version.BuildVersion, responses[0])
}

func TestTadoBot_DoRooms(t *testing.T) {
	bot := tadobot.TadoBot{
		API: &mockapi.MockAPI{},
	}

	responses := bot.DoRooms()
	sort.Strings(responses)

	assert.Equal(t, "bar: 19.9ºC (target: 25.0ºC MANUAL)", responses[0])
	assert.Equal(t, "foo: 19.9ºC (target: 20.0ºC MANUAL)", responses[1])

}

func TestTadoBot_DoUsers(t *testing.T) {
	bot := tadobot.TadoBot{
		API: &mockapi.MockAPI{},
	}

	responses := bot.DoUsers()
	sort.Strings(responses)

	assert.Equal(t, "bar: away", responses[0])
	assert.Equal(t, "foo: home", responses[1])
}
