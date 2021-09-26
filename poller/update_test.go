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

func TestUpdate_LookupZone(t *testing.T) {
	zoneID, zoneName, ok := testUpdate.LookupZone(1, "")
	assert.True(t, ok)
	assert.Equal(t, 1, zoneID)
	assert.Equal(t, "foo", zoneName)

	zoneID, zoneName, ok = testUpdate.LookupZone(0, "bar")
	assert.True(t, ok)
	assert.Equal(t, 2, zoneID)
	assert.Equal(t, "bar", zoneName)

	zoneID, zoneName, ok = testUpdate.LookupZone(0, "snafu")
	assert.False(t, ok)
}

func TestUpdate_LookupUser(t *testing.T) {
	zoneID, zoneName, ok := testUpdate.LookupUser(1, "")
	assert.True(t, ok)
	assert.Equal(t, 1, zoneID)
	assert.Equal(t, "foo", zoneName)

	zoneID, zoneName, ok = testUpdate.LookupUser(0, "bar")
	assert.True(t, ok)
	assert.Equal(t, 2, zoneID)
	assert.Equal(t, "bar", zoneName)

	zoneID, zoneName, ok = testUpdate.LookupUser(0, "snafu")
	assert.False(t, ok)

}
