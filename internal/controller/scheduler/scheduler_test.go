package scheduler_test

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScheduler_Run(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	postChannel := make(slackbot.PostChannel, 20)

	server := &mockapi.MockAPI{}
	s := scheduler.New(server, postChannel)

	go s.Run()

	s.Register <- &scheduler.Task{
		ZoneID: 2,
		State: model.ZoneState{
			State:       model.Manual,
			Temperature: tado.Temperature{Celsius: 18.5},
		},
		When: 50 * time.Millisecond,
	}

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Setting.Temperature.Celsius == 18.5
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.Register <- &scheduler.Task{
		ZoneID: 2,
		State:  model.ZoneState{State: model.Off},
		When:   10 * time.Minute,
	}

	s.Register <- &scheduler.Task{
		ZoneID: 2,
		State:  model.ZoneState{State: model.Off},
	}

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Setting.Temperature.Celsius == 5.0
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.Register <- &scheduler.Task{
		ZoneID: 2,
		State:  model.ZoneState{State: model.Auto},
	}

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Type == ""
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.Cancel <- struct{}{}

	assert.Eventually(t, func() bool {
		_, ok := <-s.Cancel
		return !ok
	}, 500*time.Millisecond, 10*time.Millisecond)

	assert.Len(t, postChannel, 7)
}
