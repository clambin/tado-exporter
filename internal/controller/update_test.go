package controller

import (
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_updateFromPollerUpdate(t *testing.T) {
	tests := []struct {
		name   string
		update poller.Update
		want   Update
	}{
		{
			name: "auto",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
			),
			want: Update{
				HomeId:     1,
				HomeState:  HomeStateAuto,
				ZoneStates: map[string]ZoneInfo{},
				Devices:    Devices{},
			},
		},
		{
			name: "auto",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME, testutils.WithPresenceLocked(false)),
			),
			want: Update{
				HomeId:     1,
				HomeState:  HomeStateAuto,
				ZoneStates: map[string]ZoneInfo{},
				Devices:    Devices{},
			},
		},
		{
			name: "home",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME, testutils.WithPresenceLocked(true)),
			),
			want: Update{
				HomeId:     1,
				HomeState:  HomeStateHome,
				ZoneStates: map[string]ZoneInfo{},
				Devices:    Devices{},
			},
		},
		{
			name: "zones",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithZone(1, "zone 1", tado.PowerON, 21, 20),
				testutils.WithZone(2, "zone 2", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithZone(3, "zone 3", tado.PowerOFF, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeTIMER, 300)),
				testutils.WithZone(4, "zone 4", tado.PowerOFF, 0, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
			),
			want: Update{
				HomeId:    1,
				HomeState: HomeStateAuto,
				ZoneStates: map[string]ZoneInfo{
					"zone 1": {ZoneStateAuto, 1},
					"zone 2": {ZoneStateManual, 2},
					"zone 3": {ZoneStateAuto, 3},
					"zone 4": {ZoneStateOff, 4},
				},
				Devices: Devices{},
			},
		},
		{
			name: "devices",
			update: testutils.Update(
				testutils.WithHome(1, "my home", tado.HOME),
				testutils.WithMobileDevice(1, "user 1"),
				testutils.WithMobileDevice(1, "user 2", testutils.WithGeoTracking()),
				testutils.WithMobileDevice(2, "user 3", testutils.WithLocation(true, false)),
				testutils.WithMobileDevice(2, "user 4", testutils.WithLocation(false, true)),
			),
			want: Update{
				HomeId:     1,
				HomeState:  HomeStateAuto,
				ZoneStates: map[string]ZoneInfo{},
				Devices: Devices{
					Device{"user 3", true},
					Device{"user 4", false},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateFromPollerUpdate(tt.update)
			assert.Equal(t, tt.want, got)
		})
	}
}
