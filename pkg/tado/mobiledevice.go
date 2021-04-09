package tado

import (
	"encoding/json"
	"fmt"
)

// MobileDevice contains the response to /api/v2/homes/<HomeID>/mobileDevices
type MobileDevice struct {
	ID       int                  `json:"id"`
	Name     string               `json:"name"`
	Settings MobileDeviceSettings `json:"settings"`
	Location MobileDeviceLocation `json:"location"`
}

// MobileDeviceSettings is a sub-structure of MobileDevice
type MobileDeviceSettings struct {
	GeoTrackingEnabled bool `json:"geoTrackingEnabled"`
}

// MobileDeviceLocation is a sub-structure of MobileDevice
type MobileDeviceLocation struct {
	Stale  bool `json:"stale"`
	AtHome bool `json:"atHome"`
}

// GetMobileDevices retrieves the status of all registered mobile devices.
func (client *APIClient) GetMobileDevices() ([]MobileDevice, error) {
	var (
		err               error
		tadoMobileDevices []MobileDevice
		body              []byte
	)
	if err = client.initialize(); err == nil {
		apiURL := client.apiURL("/mobileDevices")
		if body, err = client.call("GET", apiURL, ""); err == nil {
			err = json.Unmarshal(body, &tadoMobileDevices)
		}
	}

	return tadoMobileDevices, err
}

// String serializes a MobileDevice into a string. Used for logging.
func (mobileDevice *MobileDevice) String() string {
	return fmt.Sprintf("name=%s, geotrack=%v, stale=%v, athome=%v",
		mobileDevice.Name,
		mobileDevice.Settings.GeoTrackingEnabled,
		mobileDevice.Location.Stale,
		mobileDevice.Location.AtHome,
	)
}
