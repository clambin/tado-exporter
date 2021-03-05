package overlaylimit

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/pkg/tado"
	"time"
)

type zoneState int

const (
	zoneStateUndetermined = 0
	zoneStateAuto         = 1
	zoneStateManual       = 2
	zoneStateReported     = 3
	zoneStateExpired      = 4
)

type zoneDetails struct {
	zone        tado.Zone
	rule        configuration.OverlayLimitRule
	state       zoneState
	isOverlay   bool
	expiryTimer time.Time
}
