package setter_test

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/controller/setter"
	slackMock "github.com/clambin/tado-exporter/slackbot/mocks"
	tadoMock "github.com/clambin/tado/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func buildStack() (server *setter.Server, api *tadoMock.API, bot *slackMock.SlackBot) {
	api = &tadoMock.API{}
	bot = &slackMock.SlackBot{}
	server = setter.New(api, bot)
	return
}

func TestServer_SetOverlay(t *testing.T) {
	server, api, bot := buildStack()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run(ctx, 25*time.Millisecond)
		wg.Done()
	}()

	bot.
		On("Send", "", "good", "foo: test1", "switching off heating in 15s").
		Return(nil).
		Once()
	bot.
		On("Send", "", "good", "foo: test3", "switching off heating in 0s").
		Return(nil).
		Once()
	bot.
		On("Send", "", "good", "foo: test3", "switching off heating").
		Return(nil).
		Once()
	api.
		On("SetZoneOverlay", mock.Anything, 1, 5.0).
		Return(nil).
		Once()

	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateOff, Delay: 15 * time.Second, ActionReason: "test1"})
	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateOff, Delay: 25 * time.Second, ActionReason: "test2"})
	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateOff, Delay: 15 * time.Millisecond, ActionReason: "test3"})

	time.Sleep(100 * time.Millisecond)

	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, api, bot)
}

func TestServer_DeleteOverlay(t *testing.T) {
	server, api, bot := buildStack()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run(ctx, 25*time.Millisecond)
		wg.Done()
	}()

	bot.
		On("Send", "", "good", "foo: test1", "moving to auto mode in 15s").
		Return(nil).
		Once()
	bot.
		On("Send", "", "good", "foo: test3", "moving to auto mode in 0s").
		Return(nil).
		Once()
	bot.
		On("Send", "", "good", "foo: test3", "moving to auto mode").
		Return(nil).
		Once()
	api.
		On("DeleteZoneOverlay", mock.Anything, 1).
		Return(nil).
		Once()

	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: 15 * time.Second, ActionReason: "test1"})
	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: 25 * time.Second, ActionReason: "test2"})
	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: 25 * time.Millisecond, ActionReason: "test3"})
	time.Sleep(200 * time.Millisecond)

	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, api, bot)
}

func TestServer_Failure(t *testing.T) {
	server, api, bot := buildStack()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run(ctx, 25*time.Millisecond)
		wg.Done()
	}()

	bot.
		On("Send", "", "good", "foo: test1", "moving to auto mode in 0s").
		Return(nil).
		Once()
	bot.
		On("Send", "", "good", "foo: test1", "moving to auto mode").
		Return(nil).
		Once()
	api.
		On("DeleteZoneOverlay", mock.Anything, 1).
		Return(fmt.Errorf("API returned an error")).
		Once()
	api.
		On("DeleteZoneOverlay", mock.Anything, 1).
		Return(nil).
		Once()

	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: 25 * time.Millisecond, ActionReason: "test1"})
	time.Sleep(100 * time.Millisecond)

	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, api, bot)
}

func TestServer_Clear(t *testing.T) {
	server, api, bot := buildStack()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run(ctx, 25*time.Millisecond)
		wg.Done()
	}()

	bot.
		On("Send", "", "good", "foo: foo", "moving to auto mode in 0s").
		Return(nil).
		Once()
	bot.
		On("Send", "", "good", "foo: bar", "canceling task to move to auto mode").
		Return(nil).
		Once()

	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: 50 * time.Millisecond, ActionReason: "foo", CancelReason: "bar"})
	server.Clear(1)
	time.Sleep(100 * time.Millisecond)

	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, api, bot)
}

func TestServer_GetScheduled(t *testing.T) {
	server, _, bot := buildStack()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run(ctx, 25*time.Millisecond)
		wg.Done()
	}()

	bot.
		On("Send", "", "good", "foo: test", "switching off heating in 25m0s").
		Return(nil).
		Once()
	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateOff, Delay: 25 * time.Minute, ActionReason: "test"})

	scheduled := server.GetScheduled()
	require.Len(t, scheduled, 1)
	item, ok := scheduled[1]
	require.True(t, ok)
	assert.Equal(t, tado.ZoneState(tado.ZoneStateOff), item.State)
	assert.Equal(t, 25*time.Minute, item.Delay)
	assert.Equal(t, "test", item.ActionReason)

	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, bot)
}

func TestServer_CancelTask(t *testing.T) {
	server, api, bot := buildStack()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run(ctx, 25*time.Millisecond)
		wg.Done()
	}()

	bot.
		On("Send", "", "good", "foo: test1", "moving to auto mode in 15s").Return(nil)
	server.Set(setter.NextState{ZoneID: 1, ZoneName: "foo", State: tado.ZoneStateAuto, Delay: 15 * time.Second, ActionReason: "test1", CancelReason: "1/test1"})

	bot.
		On("Send", "", "good", "foo: 1/test1", "canceling task to move to auto mode").
		Return(nil)
	server.Clear(1)

	cancel()
	wg.Wait()

	mock.AssertExpectationsForObjects(t, api, bot)
}
