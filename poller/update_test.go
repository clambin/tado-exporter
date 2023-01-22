package poller_test

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testUpdate = poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {Name: "foo"},
			2: {Name: "bar"},
		},
	}
)

func TestUpdate_GetZoneID(t *testing.T) {
	tests := []struct {
		name string
		zone string
		pass bool
		id   int
	}{
		{
			name: "pass",
			zone: "foo",
			pass: true,
			id:   1,
		},
		{
			name: "fail",
			zone: "snafu",
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoneID, ok := testUpdate.GetZoneID(tt.zone)
			assert.Equal(t, tt.pass, ok)
			if tt.pass {
				assert.Equal(t, tt.id, zoneID)
			}
		})
	}
}

func TestUpdate_GetUserID(t *testing.T) {
	tests := []struct {
		name string
		zone string
		pass bool
		id   int
	}{
		{
			name: "pass",
			zone: "foo",
			pass: true,
			id:   1,
		},
		{
			name: "fail",
			zone: "snafu",
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoneID, ok := testUpdate.GetUserID(tt.zone)
			assert.Equal(t, tt.pass, ok)
			if tt.pass {
				assert.Equal(t, tt.id, zoneID)
			}
		})
	}
}
