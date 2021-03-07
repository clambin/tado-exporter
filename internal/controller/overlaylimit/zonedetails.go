package overlaylimit

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/pkg/tado"
	"time"
)

type zoneState int

const (
	zoneStateUndetermined = 0
	zoneStateReverted     = 1
	zoneStateAuto         = 2
	zoneStateManual       = 3
	zoneStateReported     = 4
	zoneStateExpired      = 5
)

type zoneDetails struct {
	zone        tado.Zone
	rule        configuration.OverlayLimitRule
	state       zoneState
	isOverlay   bool
	expiryTimer time.Time
}
