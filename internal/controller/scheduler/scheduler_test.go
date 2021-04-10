package scheduler_test

import (
	"github.com/clambin/tado-exporter/internal/controller/model"
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/pkg/slackbot"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScheduler_Run(t *testing.T) {
	postChannel := make(slackbot.PostChannel, 20)

	server := &mockapi.MockAPI{}
	s := scheduler.New(server, postChannel)

	go s.Run()

	s.ScheduleTask(2, model.ZoneState{State: model.Manual, Temperature: tado.Temperature{Celsius: 18.5}}, 50*time.Millisecond)

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Setting.Temperature.Celsius == 18.5
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.ScheduleTask(2, model.ZoneState{State: model.Off}, 10*time.Minute)
	s.ScheduleTask(2, model.ZoneState{State: model.Off}, 0*time.Second)

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Setting.Temperature.Celsius == 5.0
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.ScheduleTask(2, model.ZoneState{State: model.Auto}, 0*time.Second)

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Type == ""
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.Stop()

	assert.Len(t, postChannel, 7)
}

func TestScheduler_Scheduled(t *testing.T) {
	server := &mockapi.MockAPI{}
	s := scheduler.New(server, nil)

	go s.Run()

	s.ScheduleTask(2, model.ZoneState{
		State:       model.Manual,
		Temperature: tado.Temperature{Celsius: 18.5},
	},
		100*time.Millisecond,
	)

	state := s.ScheduledState(2)
	assert.Equal(t, model.Manual, state.State)

	assert.Eventually(t, func() bool {
		return s.ScheduledState(2).State == model.Unknown
	}, 200*time.Millisecond, 50*time.Millisecond)
}

func TestScheduler_Report(t *testing.T) {
	server := &mockapi.MockAPI{}
	postChannel := make(slackbot.PostChannel)
	s := scheduler.New(server, postChannel)
	go s.Run()

	s.ScheduleTask(2, model.ZoneState{State: model.Manual, Temperature: tado.Temperature{Celsius: 18.5}}, 1*time.Hour)
	_ = <-postChannel

	s.ReportTasks()

	attachments := <-postChannel
	if assert.Len(t, attachments, 1) {
		assert.Equal(t, "setting bar to 18.5º in 1h0m0s", attachments[0].Text)
	}
}
