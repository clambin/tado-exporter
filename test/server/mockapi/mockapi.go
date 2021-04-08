package mockapi

import "github.com/clambin/tado-exporter/pkg/tado"

// MockAPI mocks the tado Client API
type MockAPI struct {
	Overlays map[int]float64
}

func (client *MockAPI) GetZones() ([]*tado.Zone, error) {
	return []*tado.Zone{
		{ID: 1, Name: "foo"},
		{ID: 2, Name: "bar"},
	}, nil
}

func (client *MockAPI) GetZoneInfo(zoneID int) (*tado.ZoneInfo, error) {
	info := tado.ZoneInfo{
		Setting: tado.ZoneInfoSetting{
			Power:       "ON",
			Temperature: tado.Temperature{Celsius: 20.0},
		},
		OpenWindow: tado.ZoneInfoOpenWindow{
			DurationInSeconds:      50,
			RemainingTimeInSeconds: 250,
		},
		ActivityDataPoints: tado.ZoneInfoActivityDataPoints{
			HeatingPower: tado.Percentage{Percentage: 11.0},
		},
		SensorDataPoints: tado.ZoneInfoSensorDataPoints{
			Temperature: tado.Temperature{Celsius: 19.94},
			Humidity:    tado.Percentage{Percentage: 37.7},
		},
	}

	if temperature, ok := client.Overlays[zoneID]; ok == true {
		info.Overlay = tado.ZoneInfoOverlay{
			Type: "MANUAL",
			Setting: tado.ZoneInfoOverlaySetting{
				Type:        "HEATING",
				Power:       "ON",
				Temperature: tado.Temperature{Celsius: temperature},
			},
			Termination: tado.ZoneInfoOverlayTermination{
				Type: "MANUAL",
			},
		}
	}

	return &info, nil
}

func (client *MockAPI) GetWeatherInfo() (*tado.WeatherInfo, error) {
	return &tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 3.4},
		SolarIntensity:     tado.Percentage{Percentage: 13.3},
		WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
	}, nil
}

func (client *MockAPI) GetMobileDevices() ([]*tado.MobileDevice, error) {
	return []*tado.MobileDevice{
		{
			ID:       1,
			Name:     "foo",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: true},
		},
		{
			ID:       2,
			Name:     "bar",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: false},
		},
	}, nil
}

func (client *MockAPI) SetZoneOverlay(zoneID int, temperature float64) error {
	if client.Overlays == nil {
		client.Overlays = make(map[int]float64)
	}

	client.Overlays[zoneID] = temperature
	return nil
}

func (client *MockAPI) DeleteZoneOverlay(zoneID int) error {
	if client.Overlays == nil {
		client.Overlays = make(map[int]float64)
	}
	delete(client.Overlays, zoneID)
	return nil
}
