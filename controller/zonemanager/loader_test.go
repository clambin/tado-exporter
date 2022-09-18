package zonemanager

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/configuration"
	"github.com/clambin/tado-exporter/poller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoad(t *testing.T) {
	m := New(nil, nil, nil, config)

	update := &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			10: {ID: 10, Name: "foo"},
			11: {ID: 11, Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
			2: {ID: 2, Name: "bar"},
		},
	}

	err := m.load(update)
	require.NoError(t, err)
	assert.Equal(t, 1, m.config.ZoneID)
	assert.Equal(t, "foo", m.config.ZoneName)
	assert.Equal(t, []configuration.ZoneUser{
		{MobileDeviceID: 10, MobileDeviceName: "foo"},
	}, m.config.AutoAway.Users)

	m.loaded = false
	update = &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			11: {ID: 11, Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
			2: {ID: 2, Name: "bar"},
		},
	}

	err = m.load(update)
	require.Error(t, err)

	update = &poller.Update{
		UserInfo: map[int]tado.MobileDevice{
			10: {ID: 10, Name: "foo"},
			11: {ID: 11, Name: "bar"},
		},
		Zones: map[int]tado.Zone{
			2: {ID: 2, Name: "bar"},
		},
	}

	err = m.load(update)
	require.Error(t, err)
}
