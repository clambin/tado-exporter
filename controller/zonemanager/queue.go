package zonemanager

import (
	"context"
	"fmt"
	"github.com/clambin/tado"
	"time"
)

type Queue struct {
	tado.API
	poster Poster
	state  *NextState
}

func (q *Queue) Queue(next NextState) {
	next.When = time.Now().Add(next.Delay)

	if q.state != nil && q.state.State == next.State {
		// don't queue the state if that state is already scheduled for an earlier time
		if q.state.When.Before(next.When) {
			return
		}
	}
	q.state = &next
	if q.state.Delay > 0 {
		q.poster.NotifyQueued(*q.state)
	}
}

func (q *Queue) Clear() {
	if q.state != nil {
		q.poster.NotifyCanceled(*q.state)
	}
	q.state = nil
}

func (q *Queue) Process(ctx context.Context) (err error) {
	if q.state == nil || q.state.When.After(time.Now()) {
		return
	}

	switch q.state.State {
	case tado.ZoneStateAuto:
		err = q.API.DeleteZoneOverlay(ctx, q.state.ZoneID)
	case tado.ZoneStateOff:
		err = q.API.SetZoneOverlay(ctx, q.state.ZoneID, 5.0)
	default:
		err = fmt.Errorf("invalid queued state for zone '%s': %d", q.state.ZoneName, q.state.State)
	}

	if err == nil {
		q.poster.NotifyAction(*q.state)
		q.Clear()
	}
	return
}