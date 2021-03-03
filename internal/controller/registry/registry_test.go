package registry_test

import (
	"github.com/clambin/tado-exporter/internal/controller/registry"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegistry(t *testing.T) {
	reg := registry.Registry{
		API: &mockapi.MockAPI{},
	}

	err := reg.Run()
	assert.Nil(t, err)

	client := reg.Register()
	assert.NotNil(t, client)

	var tadoData *registry.TadoData
	if err = reg.Run(); assert.Nil(t, err) {
		if tadoData = <-client; assert.NotNil(t, tadoData) {

			assert.Len(t, tadoData.Zone, 2)
			_, ok := tadoData.Zone[1]
			assert.True(t, ok)
			_, ok = tadoData.Zone[2]
			assert.True(t, ok)

			assert.Len(t, tadoData.ZoneInfo, 2)
			_, ok = tadoData.ZoneInfo[1]
			assert.True(t, ok)
			_, ok = tadoData.ZoneInfo[2]
			assert.True(t, ok)

			assert.Len(t, tadoData.MobileDevice, 2)
			_, ok = tadoData.MobileDevice[1]
			assert.True(t, ok)
			_, ok = tadoData.MobileDevice[2]
			assert.True(t, ok)
		}
	}

	reg.Stop()
	assert.Nil(t, <-client)
}
