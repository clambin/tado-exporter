package controller

import (
	"github.com/clambin/tado-exporter/pkg/tado"
	"time"
)

// Controller object for tado-controller.
//
// Create this by supplying the necessary parameters, e.g.
//
//   rules, err := controller.ParseConfigFile("rules.yaml")
//   control := controller.Controller{
//     API:   &tado.APIClient{
//       HTTPClient: &http.Client{},
//       Username: "user@example.com",
//       Password: "somepassword",
//     },
//     Rules: rules,
//	}
type Controller struct {
	tado.API
	Rules        *Rules
	AutoAwayInfo map[int]AutoAwayInfo
	Overlays     map[int]time.Time
}

// Configuration options for tado-exporter
type Configuration struct {
	Username     string
	Password     string
	ClientSecret string
	Interval     time.Duration
	// Port         int
	Debug bool
}

// Run executes all controller rules
func (controller *Controller) Run() error {
	err := controller.runAutoAway()

	if err == nil {
		err = controller.runOverlayLimit()
	}

	return err
}

// getMobileDevice returns the mobile device matching the mobileDeviceID or mobileDeviceName from the list of mobile devices
func getMobileDevice(mobileDevices []tado.MobileDevice, mobileDeviceID int, mobileDeviceName string) *tado.MobileDevice {
	for _, mobileDevice := range mobileDevices {
		if (mobileDeviceName != "" && mobileDeviceName == mobileDevice.Name) ||
			(mobileDeviceID != 0 && mobileDeviceID == mobileDevice.ID) {
			return &mobileDevice
		}
	}

	return nil
}

// getZone returns the zone matching zoneID or zoneName from the list of zones
func getZone(zones []tado.Zone, zoneID int, zoneName string) *tado.Zone {
	for _, zone := range zones {
		if (zoneName != "" && zoneName == zone.Name) ||
			(zoneID != 0 && zoneID == zone.ID) {
			return &zone
		}
	}

	return nil

}
