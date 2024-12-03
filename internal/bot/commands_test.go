package bot

import (
	"github.com/clambin/tado-exporter/internal/bot/mocks"
	"github.com/clambin/tado-exporter/internal/poller"
	mocks2 "github.com/clambin/tado-exporter/internal/poller/mocks"
	"github.com/clambin/tado-exporter/internal/poller/testutils"
	"github.com/clambin/tado-exporter/internal/slacktools"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_commandRunner_listRooms(t *testing.T) {
	tests := []struct {
		name    string
		update  poller.Update
		wantErr assert.ErrorAssertionFunc
		want    slacktools.Attachment
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name:    "no rooms",
			update:  testutils.Update(),
			wantErr: assert.NoError,
			want:    slacktools.Attachment{Header: "Rooms:", Body: []string{"no rooms have been found"}},
		},
		{
			name: "rooms found",
			update: testutils.Update(
				testutils.WithZone(40, "room D", tado.PowerOFF, 0, 20),
				testutils.WithZone(30, "room C", tado.PowerON, 21, 20, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeTIMER, 300)),
				testutils.WithZone(20, "room B", tado.PowerON, 17.5, 21, testutils.WithZoneOverlay(tado.ZoneOverlayTerminationTypeMANUAL, 0)),
				testutils.WithZone(10, "room A", tado.PowerON, 20, 21),
			),
			wantErr: assert.NoError,
			want: slacktools.Attachment{Header: "Rooms:", Body: []string{
				"*room A*: 21.0ºC (target: 20.0)",
				"*room B*: 21.0ºC (target: 17.5, MANUAL)",
				"*room C*: 20.0ºC (target: 21.0, MANUAL for 5m0s)",
				"*room D*: 20.0ºC (off)",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := commandRunner{}
			if tt.update.HomeBase.Id != nil {
				r.setUpdate(tt.update)
			}

			got, err := r.listRooms()
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_commandRunner_listUsers(t *testing.T) {
	tests := []struct {
		name    string
		update  poller.Update
		wantErr assert.ErrorAssertionFunc
		want    slacktools.Attachment
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name:    "no users",
			update:  testutils.Update(),
			wantErr: assert.NoError,
			want:    slacktools.Attachment{Header: "Users:", Body: []string{"no users have been found"}},
		},
		{
			name: "users found",
			update: testutils.Update(
				testutils.WithMobileDevice(100, "user D", testutils.WithLocation(false, false)),
				testutils.WithMobileDevice(101, "user C", testutils.WithLocation(true, false)),
				testutils.WithMobileDevice(102, "user B", testutils.WithGeoTracking()),
				testutils.WithMobileDevice(103, "user A"),
			),
			wantErr: assert.NoError,
			want: slacktools.Attachment{Header: "Users:", Body: []string{
				"*user C*: home",
				"*user D*: away",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := commandRunner{}
			if tt.update.HomeBase.Id != nil {
				r.setUpdate(tt.update)
			}

			got, err := r.listUsers()
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_commandRunner_listRules(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.Controller)
		wantErr assert.ErrorAssertionFunc
		want    slacktools.Attachment
	}{
		{
			name:    "no update",
			wantErr: assert.Error,
		},
		{
			name: "no rules",
			setup: func(c *mocks.Controller) {
				c.EXPECT().ReportTasks().Return(nil).Once()
			},
			wantErr: assert.NoError,
			want:    slacktools.Attachment{Header: "Rules:", Body: []string{"no rules have been triggered"}},
		},
		{
			name: "rules found",
			setup: func(c *mocks.Controller) {
				c.EXPECT().ReportTasks().Return([]string{
					"room B: bar",
					"room A: foo",
				}).Once()
			},
			wantErr: assert.NoError,
			want: slacktools.Attachment{Header: "Rules:", Body: []string{
				"room A: foo",
				"room B: bar",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := commandRunner{}
			if tt.setup != nil {
				c := mocks.NewController(t)
				tt.setup(c)
				r.Controller = c
			}

			got, err := r.listRules()
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_commandRunner_refresh(t *testing.T) {
	p := mocks2.NewPoller(t)
	p.EXPECT().Refresh().Once()
	r := commandRunner{Poller: p}
	_, err := r.refresh()
	assert.NoError(t, err)
}

func Test_commandRunner_help(t *testing.T) {
	var r commandRunner
	resp, err := r.help()
	assert.NoError(t, err)
	assert.Equal(t, slacktools.Attachment{Header: "Supported commands:", Body: []string{"users, rooms, rules, help"}}, resp)
}
