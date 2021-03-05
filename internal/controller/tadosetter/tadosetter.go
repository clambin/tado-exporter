package tadosetter

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	log "github.com/sirupsen/logrus"
)

type Setter struct {
	tado.API

	ZoneSetter chan RoomCommand
	Stop       chan bool
}

type RoomCommand struct {
	ZoneID      int
	Auto        bool
	Temperature float64
}

func (setter *Setter) Run() {
loop:
	for {
		select {
		case msg := <-setter.ZoneSetter:
			setter.setRoom(msg)
		case <-setter.Stop:
			break loop
		}
	}
}

func (setter *Setter) setRoom(msg RoomCommand) {
	var err error
	if msg.Auto {
		err = setter.DeleteZoneOverlay(msg.ZoneID)
	} else {
		err = setter.SetZoneOverlay(msg.ZoneID, msg.Temperature)
	}

	if err != nil {
		log.WithField("err", err).Warning("failed to set target room temperature")
	}
}
