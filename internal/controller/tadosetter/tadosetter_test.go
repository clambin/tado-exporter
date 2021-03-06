package tadosetter_test

import (
	"github.com/clambin/tado-exporter/internal/controller/tadosetter"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetter_Run(t *testing.T) {
	server := mockapi.MockAPI{}
	setter := tadosetter.Setter{
		API:        &server,
		ZoneSetter: make(chan tadosetter.RoomCommand),
		Stop:       make(chan bool),
	}

	go setter.Run()

	setter.ZoneSetter <- tadosetter.RoomCommand{ZoneID: 1, Temperature: 15}
	setter.Stop <- true

	assert.Len(t, server.Overlays, 1)
	assert.Equal(t, 15.0, server.Overlays[1])

	go setter.Run()

	setter.ZoneSetter <- tadosetter.RoomCommand{ZoneID: 1, Auto: true}
	setter.Stop <- true

	assert.Len(t, server.Overlays, 0)
}
