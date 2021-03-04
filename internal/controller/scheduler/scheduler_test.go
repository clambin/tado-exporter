package scheduler_test

import (
	"github.com/clambin/tado-exporter/internal/controller/scheduler"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegistry(t *testing.T) {
	schedule := scheduler.Scheduler{}

	client := schedule.Register()
	assert.NotNil(t, client)

	var outData *scheduler.TadoData

	inData := scheduler.TadoData{
		Zone: map[int]tado.Zone{
			1: {},
			2: {},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {},
			2: {},
		},
		MobileDevice: map[int]tado.MobileDevice{
			1: {},
			2: {},
		},
	}

	if err := schedule.Notify(&inData); assert.Nil(t, err) {
		if outData = <-client; assert.NotNil(t, outData) {

			assert.Len(t, outData.Zone, 2)
			_, ok := outData.Zone[1]
			assert.True(t, ok)
			_, ok = outData.Zone[2]
			assert.True(t, ok)

			assert.Len(t, outData.ZoneInfo, 2)
			_, ok = outData.ZoneInfo[1]
			assert.True(t, ok)
			_, ok = outData.ZoneInfo[2]
			assert.True(t, ok)

			assert.Len(t, outData.MobileDevice, 2)
			_, ok = outData.MobileDevice[1]
			assert.True(t, ok)
			_, ok = outData.MobileDevice[2]
			assert.True(t, ok)
		}
	}

	schedule.Stop()
	assert.Nil(t, <-client)
}
