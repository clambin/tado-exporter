package autoaway

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

// DeviceInfo contains the user we are tracking, and what zone to set to which temperature
// when ActivationTime occurs
type DeviceInfo struct {
	// TODO: can this be a pointer?
	mobileDevice   tado.MobileDevice
	zone           tado.Zone
	rule           configuration.AutoAwayRule
	state          autoAwayState
	activationTime time.Time
}

func getInitialState(device *tado.MobileDevice) autoAwayState {
	if device == nil || !device.Settings.GeoTrackingEnabled || device.Location.Stale {
		return autoAwayStateUndetermined
	} else if device.Location.AtHome {
		return autoAwayStateHome
	}
	return autoAwayStateAway
}

func (autoAwayInfo *DeviceInfo) leftHome() bool {
	return autoAwayInfo.mobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.mobileDevice.Location.AtHome == false &&
		autoAwayInfo.state <= autoAwayStateHome
}

func (autoAwayInfo *DeviceInfo) cameHome() bool {
	return autoAwayInfo.mobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.mobileDevice.Location.AtHome == true &&
		(autoAwayInfo.state >= autoAwayStateAway)
}

func (autoAwayInfo *DeviceInfo) shouldReport() bool {
	return autoAwayInfo.mobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.mobileDevice.Location.AtHome == false &&
		autoAwayInfo.state < autoAwayStateExpired &&
		time.Now().After(autoAwayInfo.activationTime)
}

func (autoAwayInfo *DeviceInfo) isReported() bool {
	return autoAwayInfo.mobileDevice.Settings.GeoTrackingEnabled == true &&
		autoAwayInfo.state == autoAwayStateReported
}
