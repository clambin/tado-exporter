package scheduler_test

import (
	"github.com/clambin/tado-exporter/internal/controller/models"
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

	s.ScheduleTask(2, models.ZoneState{State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 18.5}}, 50*time.Millisecond)

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Setting.Temperature.Celsius == 18.5
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.ScheduleTask(2, models.ZoneState{State: models.ZoneOff}, 10*time.Minute)
	s.ScheduleTask(2, models.ZoneState{State: models.ZoneOff}, 0*time.Second)

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Setting.Temperature.Celsius == 5.0
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.ScheduleTask(2, models.ZoneState{State: models.ZoneAuto}, 0*time.Second)

	assert.Eventually(t, func() bool {
		if zoneInfo, err := server.GetZoneInfo(2); err == nil {
			return zoneInfo.Overlay.Type == ""
		}
		return false
	}, 500*time.Millisecond, 50*time.Millisecond)

	s.Stop()

	assert.Len(t, postChannel, 5)
}

func TestScheduler_Scheduled(t *testing.T) {
	server := &mockapi.MockAPI{}
	s := scheduler.New(server, nil)

	go s.Run()

	s.ScheduleTask(2, models.ZoneState{
		State:       models.ZoneManual,
		Temperature: tado.Temperature{Celsius: 18.5},
	},
		100*time.Millisecond,
	)

	state := s.ScheduledState(2)
	assert.Equal(t, models.ZoneManual, state.State)

	assert.Eventually(t, func() bool {
		return s.ScheduledState(2).State == models.ZoneUnknown
	}, 200*time.Millisecond, 50*time.Millisecond)
}

func TestScheduler_Report(t *testing.T) {
	server := &mockapi.MockAPI{}
	postChannel := make(slackbot.PostChannel)
	s := scheduler.New(server, postChannel)
	go s.Run()

	s.ReportTasks()

	attachments := <-postChannel
	if assert.Len(t, attachments, 1) {
		assert.Equal(t, "no rules have been triggered", attachments[0].Text)
	}

	s.ScheduleTask(2, models.ZoneState{State: models.ZoneManual, Temperature: tado.Temperature{Celsius: 18.5}}, 1*time.Hour)
	_ = <-postChannel

	s.ReportTasks()

	attachments = <-postChannel
	if assert.Len(t, attachments, 1) {
		assert.Equal(t, "bar: will set temperature to 18.5º in 1h0m0s", attachments[0].Text)
	}
}
