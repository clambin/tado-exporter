package tadoproxy_test

import (
	"github.com/clambin/tado-exporter/internal/tadoproxy"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCache_Refresh(t *testing.T) {
	proxy := tadoproxy.Proxy{
		API: &mockapi.MockAPI{},
	}

	err := proxy.Refresh()
	if assert.Nil(t, err) {
		assert.Len(t, proxy.Zone, 2)
		_, ok := proxy.Zone[1]
		assert.True(t, ok)
		_, ok = proxy.Zone[2]
		assert.True(t, ok)

		assert.Len(t, proxy.ZoneInfo, 2)
		_, ok = proxy.ZoneInfo[1]
		assert.True(t, ok)
		_, ok = proxy.ZoneInfo[2]
		assert.True(t, ok)

		assert.Len(t, proxy.MobileDevice, 2)
		_, ok = proxy.MobileDevice[1]
		assert.True(t, ok)
		_, ok = proxy.MobileDevice[2]
		assert.True(t, ok)
	}
}
