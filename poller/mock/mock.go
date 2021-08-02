package mock

import (
	"github.com/clambin/tado"
	"github.com/clambin/tado-exporter/poller"
)

var FakeUpdates = []poller.Update{
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "ON", Temperature: tado.Temperature{Celsius: 15.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones:    map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: true}}},
	},
	{
		Zones: map[int]tado.Zone{2: {ID: 2, Name: "bar"}},
		ZoneInfo: map[int]tado.ZoneInfo{2: {Setting: tado.ZoneInfoSetting{Power: "ON", Temperature: tado.Temperature{Celsius: 18.5}}, Overlay: tado.ZoneInfoOverlay{
			Type:        "MANUAL",
			Setting:     tado.ZoneInfoOverlaySetting{Type: "HEATING", Power: "OFF", Temperature: tado.Temperature{Celsius: 5.0}},
			Termination: tado.ZoneInfoOverlayTermination{Type: "MANUAL"},
		}}},
		UserInfo: map[int]tado.MobileDevice{2: {ID: 2, Name: "bar", Settings: tado.MobileDeviceSettings{GeoTrackingEnabled: true}, Location: tado.MobileDeviceLocation{AtHome: false}}},
	},
}
