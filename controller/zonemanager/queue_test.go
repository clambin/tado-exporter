package zonemanager

import (
	"context"
	"github.com/clambin/tado"
	"github.com/clambin/tado/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	api := mocks.API{}
	q := Queue{
		API:    &api,
		poster: Poster{SlackBot: nil},
	}

	_, queued := q.GetQueued()
	assert.False(t, queued)

	q.Queue(NextState{
		ZoneID:       1,
		ZoneName:     "foo",
		State:        tado.ZoneStateOff,
		Delay:        time.Hour,
		ActionReason: "action",
		CancelReason: "cancel",
	})

	_, queued = q.GetQueued()
	assert.True(t, queued)

	q.Cancel()

	_, queued = q.GetQueued()
	assert.False(t, queued)

	q.Queue(NextState{
		ZoneID:       1,
		ZoneName:     "foo",
		State:        tado.ZoneStateOff,
		Delay:        200 * time.Millisecond,
		ActionReason: "action",
		CancelReason: "cancel",
	})

	q.Queue(NextState{
		ZoneID:       1,
		ZoneName:     "foo",
		State:        tado.ZoneStateOff,
		Delay:        time.Hour,
		ActionReason: "action",
		CancelReason: "cancel",
	})

	_, queued = q.GetQueued()
	assert.True(t, queued)

	api.On("SetZoneOverlay", mock.AnythingOfType("*context.emptyCtx"), 1, 5.0).Return(nil).Once()

	assert.Eventually(t, func() bool {
		err := q.Process(context.Background())
		if err != nil {
			return false
		}
		_, queued = q.GetQueued()
		return !queued
	}, time.Second, 10*time.Millisecond)

	mock.AssertExpectationsForObjects(t, &api)

}
