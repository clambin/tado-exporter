package poller_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/clambin/tado-exporter/internal/oapi"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTadoPoller_Run(t *testing.T) {
	var client fakeClient

	p := poller.New(client, time.Minute, slog.Default())
	ch := p.Subscribe()
	go func() {
		assert.NoError(t, p.Run(t.Context()))
	}()

	update := <-ch

	assert.Equal(t, tado.HomeId(1), *update.HomeBase.Id)
	assert.True(t, update.Home())

	zone, ok := update.Zones.GetZone("room")
	require.True(t, ok)
	assert.Equal(t, 10, *zone.Zone.Id)

	device, ok := update.MobileDevices.GetMobileDevice("A")
	assert.True(t, ok)
	assert.Equal(t, tado.MobileDeviceId(100), *device.Id)
	home, away := update.MobileDevices.GetDeviceState(*device.Id)
	assert.Equal(t, []string{"A"}, home)
	assert.Equal(t, []string{}, away)

	p.Unsubscribe(ch)
}

var _ poller.TadoClient = fakeClient{}

type fakeClient struct{}

func (f fakeClient) GetMeWithResponse(_ context.Context, _ ...tado.RequestEditorFn) (*tado.GetMeResponse, error) {
	return &tado.GetMeResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &tado.User{
			Homes: &[]tado.HomeBase{{Id: oapi.VarP(tado.HomeId(1)), Name: oapi.VarP("home")}},
		},
	}, nil
}

func (f fakeClient) GetZonesWithResponse(_ context.Context, _ tado.HomeId, _ ...tado.RequestEditorFn) (*tado.GetZonesResponse, error) {
	return &tado.GetZonesResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &[]tado.Zone{{
			Id:   oapi.VarP(10),
			Name: oapi.VarP("room"),
		}},
	}, nil
}

func (f fakeClient) GetZoneStateWithResponse(_ context.Context, _ tado.HomeId, _ tado.ZoneId, _ ...tado.RequestEditorFn) (*tado.GetZoneStateResponse, error) {
	return &tado.GetZoneStateResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &tado.ZoneState{
			Setting: &tado.ZoneSetting{
				Temperature: &tado.Temperature{Celsius: oapi.VarP[float32](21)},
				Type:        oapi.VarP(tado.HEATING),
			},
		},
	}, nil
}

func (f fakeClient) GetMobileDevicesWithResponse(_ context.Context, _ tado.HomeId, _ ...tado.RequestEditorFn) (*tado.GetMobileDevicesResponse, error) {
	return &tado.GetMobileDevicesResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &[]tado.MobileDevice{{
			Id:       oapi.VarP[tado.MobileDeviceId](100),
			Name:     oapi.VarP("A"),
			Location: &oapi.LocationHome,
			Settings: &tado.MobileDeviceSettings{GeoTrackingEnabled: oapi.VarP(true)},
		}},
	}, nil
}

func (f fakeClient) GetWeatherWithResponse(_ context.Context, _ tado.HomeId, _ ...tado.RequestEditorFn) (*tado.GetWeatherResponse, error) {
	return &tado.GetWeatherResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &tado.Weather{
			OutsideTemperature: &tado.TemperatureDataPoint{Celsius: oapi.VarP[float32](19)},
			SolarIntensity:     &tado.PercentageDataPoint{Percentage: oapi.VarP[float32](75)},
			WeatherState:       &tado.WeatherStateDataPoint{Value: oapi.VarP[tado.WeatherState](tado.SUN)},
		},
	}, nil
}

func (f fakeClient) GetHomeStateWithResponse(_ context.Context, _ tado.HomeId, _ ...tado.RequestEditorFn) (*tado.GetHomeStateResponse, error) {
	return &tado.GetHomeStateResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &tado.HomeState{
			Presence:       oapi.VarP(tado.HOME),
			PresenceLocked: oapi.VarP(false),
		},
	}, nil
}
