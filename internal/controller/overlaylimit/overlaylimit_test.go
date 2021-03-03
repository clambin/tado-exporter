package overlaylimit

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/internal/controller/registry"
	"github.com/clambin/tado-exporter/pkg/tado"
	"github.com/clambin/tado-exporter/test/server/mockapi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOverlayLimit(t *testing.T) {
	tadoData := registry.TadoData{
		Zone: map[int]tado.Zone{
			1: {ID: 1, Name: "foo"},
			2: {ID: 2, Name: "bar"},
		},
		ZoneInfo: map[int]tado.ZoneInfo{
			1: {},
			2: {Overlay: tado.ZoneInfoOverlay{
				Type:        "MANUAL",
				Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING"},
				Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
			}},
		},
	}

	cfg, err := configuration.LoadConfiguration([]byte(`
controller:
  overlayLimitRules:
  - zoneName: "foo"
    maxTime: 1h
  - zoneName: "bar"
    maxTime: 1h
  - zoneName: "not-a-zone"
    maxTime: 1h
`))

	if assert.Nil(t, err) && assert.NotNil(t, cfg) && assert.NotNil(t, cfg.Controller.OverlayLimitRules) {

		server := mockapi.MockAPI{}
		reg := registry.Registry{
			API: &server,
		}

		limiter := OverlayLimit{
			Updates:  reg.Register(),
			Registry: &reg,
			Rules:    *cfg.Controller.OverlayLimitRules,
		}

		err = limiter.process(&tadoData)
		assert.Nil(t, err)

		if assert.NotNil(t, limiter.zoneDetails) {

			assert.Len(t, limiter.zoneDetails, 2)
			assert.False(t, limiter.zoneDetails[1].isOverlay)
			assert.True(t, limiter.zoneDetails[2].isOverlay)

			// zone 2 no longer on manual
			zoneInfo, _ := tadoData.ZoneInfo[2]
			zoneInfo.Overlay.Termination.Type = "TIMER"
			tadoData.ZoneInfo[2] = zoneInfo

			err = limiter.process(&tadoData)

			assert.Nil(t, err)
			assert.False(t, limiter.zoneDetails[1].isOverlay)
			assert.False(t, limiter.zoneDetails[2].isOverlay)

			// zone 2 back to manual
			zoneInfo, _ = tadoData.ZoneInfo[2]
			zoneInfo.Overlay.Termination.Type = "MANUAL"
			tadoData.ZoneInfo[2] = zoneInfo
			err = limiter.process(&tadoData)

			assert.Nil(t, err)
			assert.False(t, limiter.zoneDetails[1].isOverlay)
			assert.True(t, limiter.zoneDetails[2].isOverlay)

			// zone 2 gets expired
			details, _ := limiter.zoneDetails[2]
			details.expiryTimer = time.Now().Add(-2 * time.Hour)
			limiter.zoneDetails[2] = details

			err = limiter.process(&tadoData)

			assert.Nil(t, err)

			if assert.NotNil(t, server.Overlays) {
				_, ok := server.Overlays[2]
				assert.False(t, ok)
			}
		}

	}
}
