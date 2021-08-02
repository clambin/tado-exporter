package mockapi

import (
	"context"
	"github.com/clambin/tado"
	"sync"
	"time"
)

// MockAPI mocks the tado Client API
type MockAPI struct {
	Overlays map[int]overlaySettings
	lock     sync.RWMutex
}

type overlaySettings struct {
	temperature float64
	expiry      time.Time
}

func (client *MockAPI) GetZones(_ context.Context) ([]tado.Zone, error) {
	return []tado.Zone{
		{ID: 1, Name: "foo"},
		{ID: 2, Name: "bar"},
	}, nil
}

func (client *MockAPI) GetZoneInfo(_ context.Context, zoneID int) (info tado.ZoneInfo, err error) {
	info = tado.ZoneInfo{
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

	client.lock.RLock()
	defer client.lock.RUnlock()

	if overlay, ok := client.Overlays[zoneID]; ok == true {
		if overlay.expiry.IsZero() || time.Now().Before(overlay.expiry) {
			info.Overlay = tado.ZoneInfoOverlay{
				Type: "MANUAL",
				Setting: tado.ZoneInfoOverlaySetting{
					Type:        "HEATING",
					Power:       "ON",
					Temperature: tado.Temperature{Celsius: overlay.temperature},
				},
				Termination: tado.ZoneInfoOverlayTermination{
					Type: "MANUAL",
				},
			}

			if !overlay.expiry.IsZero() {
				info.Overlay.Termination = tado.ZoneInfoOverlayTermination{
					Type:          "TIMER",
					RemainingTime: int(time.Now().Sub(overlay.expiry)),
				}
			}
		}
	}

	return
}

func (client *MockAPI) GetWeatherInfo(_ context.Context) (tado.WeatherInfo, error) {
	return tado.WeatherInfo{
		OutsideTemperature: tado.Temperature{Celsius: 3.4},
		SolarIntensity:     tado.Percentage{Percentage: 13.3},
		WeatherState:       tado.Value{Value: "CLOUDY_MOSTLY"},
	}, nil
}

func (client *MockAPI) GetMobileDevices(_ context.Context) ([]tado.MobileDevice, error) {
	return []tado.MobileDevice{
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
		{
			ID:       3,
			Name:     "snafu",
			Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: false},
			Location: tado.MobileDeviceLocation{Stale: false, AtHome: false},
		},
	}, nil
}

func (client *MockAPI) SetZoneOverlay(_ context.Context, zoneID int, temperature float64) error {
	client.lock.Lock()
	defer client.lock.Unlock()

	if client.Overlays == nil {
		client.Overlays = make(map[int]overlaySettings)
	}

	client.Overlays[zoneID] = overlaySettings{temperature: temperature}
	return nil
}

func (client *MockAPI) SetZoneOverlayWithDuration(ctx context.Context, zoneID int, temperature float64, duration time.Duration) error {
	if duration == 0 {
		return client.SetZoneOverlay(ctx, zoneID, temperature)
	}

	client.lock.Lock()
	defer client.lock.Unlock()

	if client.Overlays == nil {
		client.Overlays = make(map[int]overlaySettings)
	}

	client.Overlays[zoneID] = overlaySettings{temperature: temperature, expiry: time.Now().Add(duration)}
	return nil
}

func (client *MockAPI) DeleteZoneOverlay(_ context.Context, zoneID int) error {
	client.lock.Lock()
	defer client.lock.Unlock()

	if client.Overlays != nil {
		delete(client.Overlays, zoneID)
	}
	return nil
}
