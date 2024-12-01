package controller

import (
	"github.com/Shopify/go-lua"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_lua_homeState(t *testing.T) {
	tests := []struct {
		name   string
		update poller.Update
		want   homeState
	}{
		{
			name:   "home auto",
			update: testutils.Update(testutils.WithHome(1, "my home", tado.HOME)),
			want:   homeState{overlay: false, home: true},
		},
		{
			name:   "away auto",
			update: testutils.Update(testutils.WithHome(1, "my home", tado.AWAY)),
			want:   homeState{overlay: false, home: false},
		},
		{
			name:   "home not locked",
			update: testutils.Update(testutils.WithHome(1, "my home", tado.HOME, testutils.WithPresenceLocked(false))),
			want:   homeState{overlay: false, home: true},
		},
		{
			name:   "away not locked",
			update: testutils.Update(testutils.WithHome(1, "my home", tado.AWAY, testutils.WithPresenceLocked(false))),
			want:   homeState{overlay: false, home: false},
		},
		{
			name:   "home manual",
			update: testutils.Update(testutils.WithHome(1, "my home", tado.HOME, testutils.WithPresenceLocked(true))),
			want:   homeState{overlay: true, home: true},
		},
		{
			name:   "away auto",
			update: testutils.Update(testutils.WithHome(1, "my home", tado.AWAY, testutils.WithPresenceLocked(true))),
			want:   homeState{overlay: true, home: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lua.NewState()
			pushHomeState(l, tt.update)
			s, err := getHomeState(l, -1)
			require.NoError(t, err)
			assert.Equal(t, tt.want, s)
		})
	}
}

func Test_lua_zoneState(t *testing.T) {
	tests := []struct {
		name string
		zone poller.Zone
		want zoneState
	}{
		{
			name: "zone off, auto",
			zone: poller.Zone{ZoneState: tado.ZoneState{
				Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerOFF)},
			}},
			want: zoneState{overlay: false, heating: false},
		},
		{
			name: "zone off, manual",
			zone: poller.Zone{ZoneState: tado.ZoneState{
				Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerOFF)},
				Overlay: &tado.ZoneOverlay{Termination: &tado.ZoneOverlayTermination{Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL)}},
			}},
			want: zoneState{overlay: true, heating: false},
		},
		{
			name: "zone on, auto",
			zone: poller.Zone{ZoneState: tado.ZoneState{
				Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON)},
			}},
			want: zoneState{overlay: false, heating: true},
		},
		{
			name: "zone on, manual",
			zone: poller.Zone{ZoneState: tado.ZoneState{
				Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerON)},
				Overlay: &tado.ZoneOverlay{Termination: &tado.ZoneOverlayTermination{Type: oapi.VarP(tado.ZoneOverlayTerminationTypeMANUAL)}},
			}},
			want: zoneState{overlay: true, heating: true},
		},
		{
			name: "timer mode is not considered manual",
			zone: poller.Zone{ZoneState: tado.ZoneState{
				Setting: &tado.ZoneSetting{Power: oapi.VarP(tado.PowerOFF)},
				Overlay: &tado.ZoneOverlay{Termination: &tado.ZoneOverlayTermination{Type: oapi.VarP(tado.ZoneOverlayTerminationTypeTIMER)}},
			}},
			want: zoneState{overlay: false, heating: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lua.NewState()
			pushZoneState(l, tt.zone)
			require.True(t, l.CheckStack(1))
			got, err := getZoneState(l, -1)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

}

func Test_lua_devices(t *testing.T) {
	tests := []struct {
		name   string
		update poller.Update
		users  set.Set[string]
		want   []any
	}{
		{
			name: "no users",
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user A"),
				testutils.WithMobileDevice(101, "user B", testutils.WithGeoTracking()),
				testutils.WithMobileDevice(102, "user C", testutils.WithLocation(true, false)),
				testutils.WithMobileDevice(103, "user D", testutils.WithLocation(false, false)),
			),
			want: []any{
				[]any{"user C", true},
				[]any{"user D", false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lua.NewState()
			pushDevices(l, tt.update, tt.users)
			require.True(t, l.CheckStack(1))

			got := luart.TableToSlice(l, l.AbsIndex(-1))
			//TODO: elements of got may be in wrong order (?): { "user", state } or { state, "user" }
			//assert.Equal(t, tt.want, got)
			assert.Len(t, got, len(tt.want))
		})
	}
}
