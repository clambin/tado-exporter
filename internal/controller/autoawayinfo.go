package controller

import (
	"github.com/clambin/tado-exporter/internal/configuration"
	"github.com/clambin/tado-exporter/pkg/tado"
	"time"
)

type autoAwayState int

const (
	autoAwayStateUndetermined = 0
	autoAwayStateHome         = 1
	autoAwayStateAway         = 2
	autoAwayStateExpired      = 3
	autoAwayStateReported     = 4
)

// AutoAwayInfo contains the user we are tracking, and what zone to set to which temperature
// when ActivationTime occurs
type AutoAwayInfo struct {
	MobileDevice *tado.MobileDevice
	state        autoAwayState
	// Home           bool
	ActivationTime time.Time
	ZoneID         int
	AutoAwayRule   *configuration.AutoAwayRule
}

func getInitialState(device *tado.MobileDevice) autoAwayState {
	if device == nil || device.Settings.GeoTrackingEnabled == false {
		return autoAwayStateUndetermined
	} else if device.Location.AtHome {
		return autoAwayStateHome
	}
	return autoAwayStateAway
}

func (autoAwayInfo *AutoAwayInfo) leftHome() bool {
	return autoAwayInfo.MobileDevice != nil &&
		autoAwayInfo.MobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.MobileDevice.Location.AtHome == false &&
		autoAwayInfo.state <= autoAwayStateHome
}

func (autoAwayInfo *AutoAwayInfo) cameHome() bool {
	return autoAwayInfo.MobileDevice != nil &&
		autoAwayInfo.MobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.MobileDevice.Location.AtHome == true &&
		(autoAwayInfo.state >= autoAwayStateAway)
}

func (autoAwayInfo *AutoAwayInfo) shouldReport() bool {
	return autoAwayInfo.MobileDevice != nil &&
		autoAwayInfo.MobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.MobileDevice.Location.AtHome == false &&
		autoAwayInfo.state < autoAwayStateExpired &&
		time.Now().After(autoAwayInfo.ActivationTime)
}

func (autoAwayInfo *AutoAwayInfo) isReported() bool {
	return autoAwayInfo.MobileDevice != nil &&
		autoAwayInfo.MobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.state == autoAwayStateReported
}

/*func (autoAwayInfo *AutoAwayInfo) nextState() (nextState autoAwayState) {
	switch autoAwayInfo.state {
	case autoAwayStateUndetermined:
		if autoAwayInfo.leftHome() {
			nextState = autoAwayStateAway
		} else {
			nextState = autoAwayStateHome
		}
	case autoAwayStateHome:
		if autoAwayInfo.cameHome() {
			nextState = autoAwayStateHome
		} else
		if autoAwayInfo.leftHome() {
			nextState = autoAwayStateExpired
		} else {
			nextState = autoAwayStateAway
		}
	case autoAwayStateAway:
		if autoAwayInfo.cameHome() {
			nextState = autoAwayStateHome
		} else
		if autoAwayInfo.shouldReport() {
			nextState = autoAwayStateExpired
		} else {
			nextState = autoAwayStateAway
		}
	case autoAwayStateExpired:
		if autoAwayInfo.cameHome() {
			nextState = autoAwayStateHome
		} else {
			nextState = autoAwayStateReported
		}
	}
	return
}
*/
